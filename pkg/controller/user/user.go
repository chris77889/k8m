package user

import (
	"encoding/base64"
	"fmt"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/k8m/internal/dao"
	"github.com/weibaohui/k8m/pkg/comm/utils"
	"github.com/weibaohui/k8m/pkg/comm/utils/amis"
	"github.com/weibaohui/k8m/pkg/models"
	"gorm.io/gorm"
)

// 获取当前用户的Role信息
func Role(c *gin.Context) {
	_, role := amis.GetLoginUser(c)
	amis.WriteJsonData(c, gin.H{
		"role": role,
	})
}
func List(c *gin.Context) {
	params := dao.BuildParams(c)
	m := &models.User{}

	queryFuncs := genQueryFuncs(c, params)
	queryFuncs = append(queryFuncs, func(db *gorm.DB) *gorm.DB {
		return db.Select([]string{"id", "group_names", "two_fa_enabled", "username", "two_fa_type", "two_fa_app_name", "created_at", "updated_at"})
	})
	items, total, err := m.List(params, queryFuncs...)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonListWithTotal(c, total, items)
}
func Save(c *gin.Context) {
	params := dao.BuildParams(c)
	m := &models.User{}
	err := c.ShouldBindJSON(&m)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	// 用户名不能为admin
	if m.Username == "admin" {
		amis.WriteJsonError(c, fmt.Errorf("用户名不能为admin"))
		return
	}

	_, role := amis.GetLoginUser(c)

	if m.ID == 0 {
		// 新增
		switch role {
		case models.RoleClusterAdmin, models.RoleClusterReadonly:
			amis.WriteJsonError(c, fmt.Errorf("非管理员不能新增用户"))
			return
		}
	} else {
		switch role {
		case models.RoleClusterAdmin, models.RoleClusterReadonly:
			var originalUser models.User
			err = dao.DB().Model(&models.User{}).
				Where("id=?", m.ID).
				Find(&originalUser).Error
			if err != nil {
				amis.WriteJsonError(c, fmt.Errorf("无此用户[%d]", m.ID))
				return
			}

			// 如需限制不能修改的字段，请在下面赋值。
			// 用户名、角色不能修改
			m.Username = originalUser.Username
			m.GroupNames = originalUser.GroupNames
		}

	}

	queryFuncs := genQueryFuncs(c, params)

	// 保存的时候需要单独处理
	queryFuncs = append(queryFuncs, func(db *gorm.DB) *gorm.DB {
		if m.ID == 0 {
			// 新增
			return db
		} else {
			// 修改
			return db.Select([]string{"username", "group_names"})
		}
	})
	err = m.Save(params, queryFuncs...)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonData(c, gin.H{
		"id": m.ID,
	})
}
func Delete(c *gin.Context) {
	ids := c.Param("ids")
	params := dao.BuildParams(c)
	m := &models.User{}

	_, role := amis.GetLoginUser(c)

	switch role {
	case models.RoleClusterReadonly, models.RoleClusterAdmin:
		// 非平台管理员，不能删除
		amis.WriteJsonError(c, fmt.Errorf("非管理员不能删除用户"))
		return
	}

	queryFuncs := genQueryFuncs(c, params)

	err := m.Delete(params, ids, queryFuncs...)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}
func UpdatePsw(c *gin.Context) {

	id := c.Param("id")
	params := dao.BuildParams(c)
	m := &models.User{}
	err := c.ShouldBindJSON(m)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	m.ID = uint(utils.ToInt64(id))

	// 密码 + 盐重新计算
	pswBytes, err := utils.AesDecrypt(m.Password)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	m.Salt = utils.RandNLengthString(8)
	psw, err := utils.AesEncrypt([]byte(fmt.Sprintf("%s%s", string(pswBytes), m.Salt)))
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	m.Password = base64.StdEncoding.EncodeToString(psw)

	queryFuncs := genQueryFuncs(c, params)
	queryFuncs = append(queryFuncs, func(db *gorm.DB) *gorm.DB {
		return db.Select([]string{"password", "salt"}).Updates(m)
	})
	err = m.Save(params, queryFuncs...)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}

func genQueryFuncs(c *gin.Context, params *dao.Params) []func(*gorm.DB) *gorm.DB {
	//  管理页面，判断是否管理员，看到所有的用户，
	user, role := amis.GetLoginUser(c)
	var queryFuncs []func(*gorm.DB) *gorm.DB
	switch role {
	case models.RolePlatformAdmin:
		params.UserName = ""
		queryFuncs = []func(*gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				return db
			},
		}
	case models.RoleClusterAdmin, models.RoleClusterReadonly:
		queryFuncs = []func(*gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				return db.Where("username=?", user)
			},
		}

	}
	return queryFuncs
}

func UserOptionList(c *gin.Context) {
	params := dao.BuildParams(c)
	m := &models.User{}
	items, _, err := m.List(params, func(db *gorm.DB) *gorm.DB {
		return db.Distinct("username")
	})
	if err != nil {
		amis.WriteJsonData(c, gin.H{
			"options": make([]map[string]string, 0),
		})
		return
	}
	var names []map[string]string
	for _, n := range items {
		names = append(names, map[string]string{
			"label": n.Username,
			"value": n.Username,
		})
	}
	slice.SortBy(names, func(a, b map[string]string) bool {
		return a["label"] < b["label"]
	})
	amis.WriteJsonData(c, gin.H{
		"options": names,
	})
}
