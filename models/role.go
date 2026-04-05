package models

import (
	"encoding/json"
	"finance/database"
	"time"
)

type RoleName string

const (
	RoleViewer RoleName="viewer"
	RoleAnalyst RoleName="analyst"
	RoleAdmin RoleName="admin"
)

type Permission struct {
	CanViewDashboard bool `json:"can_view_dashboard"`
	CanViewRecords   bool `json:"can_view_records"`
	CanViewInsights  bool `json:"can_view_insights"`
	CanCreateRecords bool `json:"can_create_records"`
	CanUpdateRecords bool `json:"can_update_records"`
	CanDeleteRecords bool `json:"can_delete_records"`
	CanManageUsers   bool `json:"can_manage_users"`
}

type Role struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:50;unique;not null"`
	Permissions string    `gorm:"type:json"`
	CreatedAt   time.Time `gorm:"column:date_added;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:last_modified;autoUpdateTime"`
}

func (r *Role) Create() error{
	return database.Database.Db.Create(r).Error
}

func (r *Role) GetById(id uint) error{
	return database.Database.Db.First(r,id).Error
}

func (r *Role) GetByName(name string) error{
	return database.Database.Db.Where("name = ?",name).First(r).Error
}

func (r *Role) Update() error{
	return database.Database.Db.Save(r).Error
}

func (r *Role) Delete() error{
	return database.Database.Db.Delete(r).Error
}

func (r *Role) GetPermissions()(Permission,error){
	var perm Permission

	if r.Permissions==""{
		return perm,nil
	}

	err:=json.Unmarshal([]byte(r.Permissions),&perm)
	return perm,err
}

func SeedRoles()error{
	roles := []struct {
		Name RoleName
		Perm Permission
	}{
		{
			Name: RoleViewer,
			Perm: Permission{
				CanViewDashboard: true,
			},
		},
		{
			Name: RoleAnalyst,
			Perm: Permission{
				CanViewDashboard: true,
				CanViewRecords:   true,
				CanViewInsights:  true,
			},
		},
		{
			Name: RoleAdmin,
			Perm: Permission{
				CanViewDashboard: true,
				CanViewRecords:   true,
				CanViewInsights:  true,
				CanCreateRecords: true,
				CanUpdateRecords: true,
				CanDeleteRecords: true,
				CanManageUsers:   true,
			},
		},
	}

	for _,r:=range roles{
		bytes,_:=json.Marshal(r.Perm)

		role:=Role{
			Name: string(r.Name),
			Permissions: string(bytes),
		}

		if err:=database.Database.Db.Where("name=?",r.Name).FirstOrCreate(&role).Error;err!=nil{
			return err
		}
	}
	return nil
}