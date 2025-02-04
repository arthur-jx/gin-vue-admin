package service

import (
	"database/sql"
	"fmt"
	"gin-vue-admin/config"
	"gin-vue-admin/global"
	"gin-vue-admin/model"
	"gin-vue-admin/model/request"
	"gin-vue-admin/source"
	"gin-vue-admin/utils"
	"path/filepath"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//@author: [songzhibin97](https://github.com/songzhibin97)
//@function: writeConfig
//@description: 回写配置
//@param: viper *viper.Viper, mysql config.Mysql
//@return: error

func writeConfig(viper *viper.Viper, mysql config.Mysql) error {
	global.GVA_CONFIG.Mysql = mysql
	cs := utils.StructToMap(global.GVA_CONFIG)
	for k, v := range cs {
		viper.Set(k, v)
	}
	return viper.WriteConfig()
}

func writeSqliteConfig(viper *viper.Viper, sqlite config.Sqlite) error {
	global.GVA_CONFIG.Sqlite = sqlite
	cs := utils.StructToMap(global.GVA_CONFIG)
	for k, v := range cs {
		viper.Set(k, v)
	}
	return viper.WriteConfig()
}

//@author: [songzhibin97](https://github.com/songzhibin97)
//@function: createTable
//@description: 创建数据库(mysql)
//@param: dsn string, driver string, createSql
//@return: error

func createTable(dsn string, driver string, createSql string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)
	if err = db.Ping(); err != nil {
		return err
	}
	_, err = db.Exec(createSql)
	return err
}

func initDB(InitDBFunctions ...model.InitDBFunc) (err error) {
	for _, v := range InitDBFunctions {
		err = v.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

//@author: [songzhibin97](https://github.com/songzhibin97)
//@function: InitDB
//@description: 创建数据库并初始化
//@param: conf request.InitDB
//@return: error

func InitDB(conf request.InitDB) error {
	BaseMysql := config.Mysql{
		Path:     "",
		Dbname:   "",
		Username: "",
		Password: "",
		Config:   "charset=utf8mb4&parseTime=True&loc=Local",
	}

	BaseSqlite := config.Sqlite{
		Path:   "",
		Dbname: "",
		Config: "",
	}

	if conf.SqlType == "mysql" {

		if conf.Host == "" {
			conf.Host = "127.0.0.1"
		}

		if conf.Port == "" {
			conf.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", conf.UserName, conf.Password, conf.Host, conf.Port)
		createSql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci;", conf.DBName)
		if err := createTable(dsn, "mysql", createSql); err != nil {
			return err
		}

		MysqlConfig := config.Mysql{
			Path:     fmt.Sprintf("%s:%s", conf.Host, conf.Port),
			Dbname:   conf.DBName,
			Username: conf.UserName,
			Password: conf.Password,
			Config:   "charset=utf8mb4&parseTime=True&loc=Local",
		}

		if err := writeConfig(global.GVA_VP, MysqlConfig); err != nil {
			return err
		}
		m := global.GVA_CONFIG.Mysql
		if m.Dbname == "" {
			return nil
		}

		linkDns := m.Username + ":" + m.Password + "@tcp(" + m.Path + ")/" + m.Dbname + "?" + m.Config
		mysqlConfig := mysql.Config{
			DSN:                       linkDns, // DSN data source name
			DefaultStringSize:         191,     // string 类型字段的默认长度
			DisableDatetimePrecision:  true,    // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
			DontSupportRenameIndex:    true,    // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
			DontSupportRenameColumn:   true,    // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
			SkipInitializeWithVersion: false,   // 根据版本自动配置
		}
		if db, err := gorm.Open(mysql.New(mysqlConfig), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}); err != nil {
			//global.GVA_LOG.Error("MySQL启动异常!", zap.Any("err", err))
			//os.Exit(0)
			//return nil
			_ = writeConfig(global.GVA_VP, BaseMysql)
			return nil
		} else {
			sqlDB, _ := db.DB()
			sqlDB.SetMaxIdleConns(m.MaxIdleConns)
			sqlDB.SetMaxOpenConns(m.MaxOpenConns)
			global.GVA_DB = db
		}
	} else if conf.SqlType == "sqlite" {
		sqliteConfig := config.Sqlite{
			Path:   conf.Path,
			Dbname: conf.DBName,
			Config: "",
		}

		if err := writeSqliteConfig(global.GVA_VP, sqliteConfig); err != nil {
			return err
		}
		m := global.GVA_CONFIG.Sqlite
		if m.Dbname == "" {
			return nil
		}

		dbpath := conf.DBName
		if len(conf.Path) > 0 {
			dbpath = fmt.Sprintf("%s/%s", conf.Path, conf.DBName)
		}

		if db, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{}); err != nil {
			_ = writeSqliteConfig(global.GVA_VP, BaseSqlite)
			return nil
		} else {
			global.GVA_DB = db
		}
	}

	err := global.GVA_DB.AutoMigrate(
		model.SysUser{},
		model.SysAuthority{},
		model.SysApi{},
		model.SysBaseMenu{},
		model.SysBaseMenuParameter{},
		model.JwtBlacklist{},
		model.SysDictionary{},
		model.SysDictionaryDetail{},
		model.ExaFileUploadAndDownload{},
		model.ExaFile{},
		model.ExaFileChunk{},
		model.ExaSimpleUploader{},
		model.ExaCustomer{},
		model.SysOperationRecord{},
	)
	if err != nil {
		if conf.SqlType == "sqlite" {
			_ = writeSqliteConfig(global.GVA_VP, BaseSqlite)
		} else {
			_ = writeConfig(global.GVA_VP, BaseMysql)
		}
		return err
	}
	err = initDB(
		source.Admin,
		source.Api,
		source.AuthorityMenu,
		source.Authority,
		source.AuthoritiesMenus,
		source.Casbin,
		source.DataAuthorities,
		source.Dictionary,
		source.DictionaryDetail,
		source.File,
		source.BaseMenu)
	if err != nil {
		if conf.SqlType == "sqlite" {
			_ = writeSqliteConfig(global.GVA_VP, BaseSqlite)
		} else {
			_ = writeConfig(global.GVA_VP, BaseMysql)
		}
		return err
	}
	global.GVA_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	return nil
}
