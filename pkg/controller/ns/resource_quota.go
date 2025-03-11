package ns

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/k8m/pkg/comm/utils/amis"
	"github.com/weibaohui/kom/kom"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateResouceQuota(c *gin.Context) {
	ctx := amis.GetContextWithUser(c)
	selectedCluster := amis.GetSelectedCluster(c)

	var data struct {
		Name     string `json:"name"`
		Metadata struct {
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Hard struct {
				Requests               map[string]string `json:"requests"`
				Limits                 map[string]string `json:"limits"`
				Pods                   string            `json:"pods"`
				Configmaps             string            `json:"configmaps"`
				Replicationcontrollers string            `json:"replicationcontrollers"`
				Resourcequotas         string            `json:"resourcequotas"`
				Services               string            `json:"services"`
				Loadbalancers          string            `json:"loadbalancers"`
				Nodeports              string            `json:"nodeports"`
				Secrets                string            `json:"secrets"`
				Persistentvolumeclaims string            `json:"persistentvolumeclaims"`
			} `json:"hard"`
		} `json:"spec"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	quota := &v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.Metadata.Namespace,
		},
		Spec: v1.ResourceQuotaSpec{
			Hard: make(v1.ResourceList),
		},
	}

	// 处理requests资源
	for name, value := range data.Spec.Hard.Requests {
		if name == "cpu" {
			value = fmt.Sprintf("%sm", value)
		}
		if name == "memory" || name == "storage" {
			value = fmt.Sprintf("%sGi", value)
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			amis.WriteJsonError(c, err)
			return
		}
		quota.Spec.Hard[v1.ResourceName("requests."+name)] = quantity
	}

	// 处理limits资源
	for name, value := range data.Spec.Hard.Limits {
		if name == "cpu" {
			value = fmt.Sprintf("%sm", value)
		}
		if name == "memory" || name == "storage" {
			value = fmt.Sprintf("%sGi", value)
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			amis.WriteJsonError(c, err)
			return
		}
		quota.Spec.Hard[v1.ResourceName("limits."+name)] = quantity
	}

	// 处理其他资源
	resourceMap := map[string]string{
		"pods":                   data.Spec.Hard.Pods,
		"configmaps":             data.Spec.Hard.Configmaps,
		"replicationcontrollers": data.Spec.Hard.Replicationcontrollers,
		"resourcequotas":         data.Spec.Hard.Resourcequotas,
		"services":               data.Spec.Hard.Services,
		"services.loadbalancers": data.Spec.Hard.Loadbalancers,
		"services.nodeports":     data.Spec.Hard.Nodeports,
		"secrets":                data.Spec.Hard.Secrets,
		"persistentvolumeclaims": data.Spec.Hard.Persistentvolumeclaims,
	}

	for name, value := range resourceMap {
		if value != "" {
			quantity, err := resource.ParseQuantity(value)
			if err != nil {
				amis.WriteJsonError(c, err)
				return
			}
			quota.Spec.Hard[v1.ResourceName(name)] = quantity
		}
	}

	err := kom.Cluster(selectedCluster).WithContext(ctx).
		Resource(quota).
		Name(data.Name).
		Namespace(data.Metadata.Namespace).
		Create(&quota).Error

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}
