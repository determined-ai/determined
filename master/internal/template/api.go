package template

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// RegisterAPIHandler initializes and registers the API handlers for all template related features.
func RegisterAPIHandler(echo *echo.Echo, db *db.PgDB, middleware ...echo.MiddlewareFunc) {
	m := &manager{db: db}
	apiGroup := echo.Group("/templates", middleware...)
	apiGroup.GET("", api.Route(m.list))
	apiGroup.GET("/:template_name", api.Route(m.get))
	apiGroup.PUT("/:template_name", api.Route(m.put))
	apiGroup.DELETE("/:template_name", api.Route(m.delete))
}

type manager struct{ db *db.PgDB }

func (m *manager) list(c echo.Context) (interface{}, error) {
	return m.db.TemplateList()
}

func (m *manager) get(c echo.Context) (interface{}, error) {
	return m.db.TemplateByName(c.Param("template_name"))
}

func (m *manager) put(c echo.Context) (interface{}, error) {
	args := struct {
		Name string `path:"template_name"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	name := args.Name
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(body, make(map[interface{}]interface{})); err != nil {
		return nil, errors.Wrap(err, "invalid YAML for template")
	}
	return nil, errors.Wrapf(
		m.db.UpsertTemplate(&model.Template{Name: name, Config: body}),
		"error putting template %q", name)
}

func (m *manager) delete(c echo.Context) (interface{}, error) {
	args := struct {
		Name string `path:"template_name"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	name := args.Name
	if err := m.db.DeleteTemplate(name); err != nil {
		return nil, errors.Wrapf(err, "deleting template %q", name)
	}
	return nil, nil
}
