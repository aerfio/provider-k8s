package clientregistry

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type Placeholder struct {
}

func (p Placeholder) Register(c controller.Controller) {

}
