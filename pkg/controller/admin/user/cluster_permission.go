package user

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/k8m/internal/dao"
	"github.com/weibaohui/k8m/pkg/comm/utils"
	"github.com/weibaohui/k8m/pkg/comm/utils/amis"
	"github.com/weibaohui/k8m/pkg/constants"
	"github.com/weibaohui/k8m/pkg/models"
	"github.com/weibaohui/k8m/pkg/service"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

func ListClusterPermissions(c *gin.Context) {
	clusterBase64 := c.Param("cluster")
	role := c.Param("role")
	cluster, err := utils.DecodeBase64(clusterBase64)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	params := dao.BuildParams(c)
	m := &models.ClusterUserRole{}
	m.Cluster = cluster
	m.Role = role
	queryFuncs := genQueryFuncs(c, params)
	queryFuncs = append(queryFuncs, func(db *gorm.DB) *gorm.DB {
		return db.Where(m).Order("authorization_type desc ,username asc")
	})
	items, total, err := m.List(params, queryFuncs...)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonListWithTotal(c, total, items)
}
func ListClusterPermissionsByUserName(c *gin.Context) {
	username := c.Param("username")
	params := dao.BuildParams(c)
	params.PerPage = 10000
	m := &models.ClusterUserRole{}
	m.Username = username
	items, total, err := m.List(params, func(db *gorm.DB) *gorm.DB {
		return db.Where(m).Order("authorization_type desc ,username asc")
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonListWithTotal(c, total, items)
}
func ListClusterPermissionsByClusterID(c *gin.Context) {
	clusterBase64 := c.Param("cluster")
	cluster, err := utils.DecodeBase64(clusterBase64)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	params := dao.BuildParams(c)
	m := &models.ClusterUserRole{}
	m.Cluster = cluster
	items, total, err := m.List(params, func(db *gorm.DB) *gorm.DB {
		return db.Where(m).Order("authorization_type desc ,username asc")
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonListWithTotal(c, total, items)
}
func SaveClusterPermission(c *gin.Context) {
	clusterBase64 := c.Param("cluster")
	role := c.Param("role")
	authorizationType := c.Param("authorization_type")
	cluster, err := utils.DecodeBase64(clusterBase64)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	// {"users":"lisi,no2fa,test"}
	type requestBody struct {
		Users string `json:"users"`
	}
	var userList requestBody

	err = c.ShouldBindJSON(&userList)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	if userList.Users == "" {
		amis.WriteJsonError(c, fmt.Errorf("用户列表不能为空"))
		return
	}

	_, _, err = handlePermissionCommonLogic(c, "保存", cluster, gin.H{
		"users":   userList.Users,
		"role":    role,
		"cluster": cluster,
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	params := dao.BuildParams(c)

	// 默认授权类型为用户
	if authorizationType == "" {
		authorizationType = "user"
	}
	for _, username := range strings.Split(userList.Users, ",") {
		var m models.ClusterUserRole
		m.Cluster = cluster
		m.Role = role
		m.Username = username
		m.AuthorizationType = constants.ClusterAuthorizationType(authorizationType)
		one, err := m.GetOne(params, func(db *gorm.DB) *gorm.DB {
			return db.Where(m)
		})

		if err != nil || one == nil {
			// 不在用户权限条目，则添加
			err := m.Save(params)
			if err != nil {
				klog.V(6).Infof("新增用户权限失败: %s", err.Error())
				continue
			}
		}

		// 如果存在该集群下的用户条目，则跳过，不做处理
	}

	amis.WriteJsonOK(c)
}

func DeleteClusterPermission(c *gin.Context) {
	ids := c.Param("ids")

	_, _, err := handlePermissionCommonLogic(c, "删除", "", gin.H{"ids": ids})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	params := dao.BuildParams(c)
	m := &models.ClusterUserRole{}

	err = m.Delete(params, ids)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}

func UpdateNamespaces(c *gin.Context) {
	id := c.Param("id")
	type requestBody struct {
		Namespaces []string `json:"namespaces"`
		Username   string   `json:"username"`
		Cluster    string   `json:"cluster"`
		Role       string   `json:"role"`
	}
	var nsList requestBody

	err := c.ShouldBindJSON(&nsList)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	_, _, err = handlePermissionCommonLogic(c, "授权Namespace", nsList.Cluster, gin.H{
		"request": nsList,
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	params := dao.BuildParams(c)
	m := &models.ClusterUserRole{}
	m.ID = utils.ToUInt(id)
	m.Namespaces = strings.Join(nsList.Namespaces, ",")
	err = m.Save(params, func(db *gorm.DB) *gorm.DB {
		return db.Select("namespaces")
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}

// TODO 日志记录写一个专门的方法，现在这个不好
func log2DB(c *gin.Context, action string, clusterName string, params gin.H, err error) {
	username, role := amis.GetLoginUser(c)
	log := models.OperationLog{
		Action:       action,
		Cluster:      clusterName,
		Kind:         "ClusterPermission",
		Name:         clusterName,
		UserName:     username,
		Group:        clusterName,
		Role:         role,
		ActionResult: "success",
	}

	if err != nil {
		log.ActionResult = err.Error()
	}

	service.OperationLogService().Add(&log, params)
}
func handlePermissionCommonLogic(c *gin.Context, action string, clusterName string, params gin.H) (string, string, error) {
	username, role := amis.GetLoginUser(c)
	var err error
	if !amis.IsCurrentUserPlatformAdmin(c) {
		err = fmt.Errorf("非平台管理员不能%s权限配置", action)
	}
	log2DB(c, action, clusterName, params, err)

	return username, role, err
}
