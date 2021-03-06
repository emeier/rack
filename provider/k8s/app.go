package k8s

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/apimachinery/pkg/api/errors"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	if _, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(name), am.GetOptions{}); !ae.IsNotFound(err) {
		return nil, fmt.Errorf("app already exists: %s", name)
	}

	params := map[string]interface{}{
		"Name":      name,
		"Namespace": p.AppNamespace(name),
		"Rack":      p.Rack,
	}

	if out, err := p.ApplyTemplate("app", fmt.Sprintf("system=convox,provider=k8s,scope=app,rack=%s,app=%s", p.Rack, name), params); err != nil {
		return nil, fmt.Errorf("create error: %s", string(out))
	}

	return p.AppGet(name)
}

func (p *Provider) AppDelete(name string) error {
	if _, err := p.AppGet(name); err != nil {
		return err
	}

	if err := p.Cluster.CoreV1().Namespaces().Delete(p.AppNamespace(name), nil); err != nil {
		return err
	}

	// if err := p.Storage.Clear(fmt.Sprintf("apps/%s", name)); err != nil {
	//   return err
	// }

	return nil
}

func (p *Provider) AppGet(name string) (*structs.App, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(name), am.GetOptions{})
	if ae.IsNotFound(err) {
		return nil, fmt.Errorf("app not found: %s", name)
	}
	if err != nil {
		return nil, err
	}

	a := appFromNamespace(*ns)

	return &a, nil
}

func (p *Provider) AppList() (structs.Apps, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().List(am.ListOptions{
		LabelSelector: fmt.Sprintf("system=convox,rack=%s,type=app", p.Rack),
	})
	if err != nil {
		return nil, err
	}

	// fmt.Printf("ns = %+v\n", ns)

	as := structs.Apps{}

	for _, n := range ns.Items {
		as = append(as, appFromNamespace(n))
	}

	return as, nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppMetrics(name string, opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppNamespace(app string) string {
	switch app {
	case "system":
		return "convox-system"
	case "rack":
		return p.Rack
	default:
		return fmt.Sprintf("%s-%s", p.Rack, app)
	}
}

func (p *Provider) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}

func appFromNamespace(ns ac.Namespace) structs.App {
	status := "unknown"

	switch ns.Status.Phase {
	case "Active":
		status = "running"
	case "Terminating":
		status = "deleting"
	}

	return structs.App{
		Generation: "2",
		Name:       helpers.CoalesceString(ns.Labels["app"], ns.Labels["name"]),
		Release:    ns.Annotations["convox.release"],
		Status:     status,
	}
}
