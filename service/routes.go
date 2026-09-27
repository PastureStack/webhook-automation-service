package service

import (
	"crypto/rsa"
	"net/http"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/sirupsen/logrus"
)

var schemas *v1client.Schemas

func HandleError(s *v1client.Schemas, t func(http.ResponseWriter, *http.Request) (int, error)) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if code, err := t(rw, req); err != nil {
			apiContext := api.GetApiContext(req)
			logrus.Errorf("Error in request: %v", err)
			rw.Header().Add("Content-Type", "application/json")
			rw.WriteHeader(code)
			writeErr := apiContext.WriteResource(&model.ServerAPIError{
				Resource: v1client.Resource{
					Type: "error",
				},
				Code:    code,
				Status:  "Server Error",
				Message: err.Error(),
			})
			if writeErr != nil {
				logrus.Errorf("Failed to write err: %v", err)
			}
		} else {
			if code != 200 {
				rw.WriteHeader(code)
			}
		}
	}))
}

type RouteHandler struct {
	ClientFactory APIClientFactory
	PublicKey     *rsa.PublicKey
}

func NewRouter(r *RouteHandler) *mux.Router {
	schemas = driverSchemas()
	router := mux.NewRouter().StrictSlash(false)
	f := HandleError

	router.Methods("GET").Path("/v1-webhooks").Handler(VersionHandler(schemas))
	router.Methods("GET").Path("/v1-webhooks/").Handler(VersionHandler(schemas))

	router.Methods("GET").Path("/v1-webhooks/schemas/").Handler(roleAwareSchemasHandler(schemas, false))
	router.Methods("GET").Path("/v1-webhooks/schemas").Handler(roleAwareSchemasHandler(schemas, false))

	router.Methods("GET").Path("/v1-webhooks/schemas/{id}").Handler(roleAwareSchemasHandler(schemas, true))
	router.Methods("GET").Path("/v1-webhooks/schemas/{id}/").Handler(roleAwareSchemasHandler(schemas, true))

	router.Methods("POST").Path("/v1-webhooks/receivers").Handler(f(schemas, r.ConstructPayload))
	router.Methods("POST").Path("/v1-webhooks/receivers/").Handler(f(schemas, r.ConstructPayload))

	router.Methods("GET").Path("/v1-webhooks/receivers").Handler(f(schemas, r.ListWebhooks))
	router.Methods("GET").Path("/v1-webhooks/receivers/").Handler(f(schemas, r.ListWebhooks))

	router.Methods("GET").Path("/v1-webhooks/receivers/{id}").Handler(f(schemas, r.GetWebhook))
	router.Methods("GET").Path("/v1-webhooks/receivers/{id}/").Handler(f(schemas, r.GetWebhook))

	router.Methods("DELETE").Path("/v1-webhooks/receivers/{id}").Handler(f(schemas, r.DeleteWebhook))
	router.Methods("DELETE").Path("/v1-webhooks/receivers/{id}/").Handler(f(schemas, r.DeleteWebhook))

	router.Methods("POST").Path("/v1-webhooks/endpoint").Handler(f(schemas, r.Execute))
	router.Methods("POST").Path("/v1-webhooks/endpoint/").Handler(f(schemas, r.Execute))

	return router
}

// The receiver write boundary depends on the authenticated project's roles.
// Advertise exactly that boundary to clients, without mutating the shared
// schema (the Rancher schema handler adds links while rendering a response).
func roleAwareSchemasHandler(base *v1client.Schemas, single bool) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Capabilities vary by the authenticated project role; a shared cache
		// must never serve an owner's writable schema to a read-only user.
		rw.Header().Set("Cache-Control", "private, no-store")
		rw.Header().Add("Vary", RoleAPIHeader)
		scoped := *base
		scoped.Data = make([]v1client.Schema, len(base.Data))
		readonly := hasReadonlyRole(r)
		for i, original := range base.Data {
			copy := original
			copy.Links = make(map[string]string, len(original.Links))
			for key, value := range original.Links {
				copy.Links[key] = value
			}
			if copy.Id == "receiver" && readonly {
				copy.CollectionMethods = []string{"GET"}
				copy.ResourceMethods = []string{"GET"}
			}
			scoped.Data[i] = copy
		}
		if single {
			api.SchemaHandler(&scoped).ServeHTTP(rw, r)
		} else {
			api.SchemasHandler(&scoped).ServeHTTP(rw, r)
		}
	})
}

func driverSchemas() *v1client.Schemas {
	schemas := &v1client.Schemas{}
	webhook := schemas.AddType("receiver", model.Webhook{})
	webhook.CollectionMethods = []string{"GET", "POST"}
	webhook.ResourceMethods = []string{"GET", "DELETE"}

	f := webhook.ResourceFields["name"]
	f.Create = true
	webhook.ResourceFields["name"] = f

	driverOptions := []string{}
	for key, value := range drivers.Drivers {
		webhookField := key + "Config"
		if field, ok := webhook.ResourceFields[webhookField]; ok {
			driverOptions = append(driverOptions, key)
			field.Type = key
			field.Create = true
			webhook.ResourceFields[webhookField] = field
			driverConfig := schemas.AddType(key, value.GetDriverConfigResource())
			driverConfig.CollectionMethods = []string{}
			for k, f := range driverConfig.ResourceFields {
				f.Create = true
				driverConfig.ResourceFields[k] = f
			}
			driverConfig = value.CustomizeSchema(driverConfig)
		} else {
			logrus.Warnf("Skipping configured driver %v because it doesn't have a field on webhook", key)
		}
	}

	f = webhook.ResourceFields["driver"]
	f.Create = true
	f.Type = "enum"
	f.Options = driverOptions
	webhook.ResourceFields["driver"] = f

	schemas.AddType("apiVersion", v1client.Resource{})
	schemas.AddType("schema", v1client.Schema{})
	schemas.AddType("error", model.ServerAPIError{})

	return schemas
}
