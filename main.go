package main

import (
	"fmt"
	"net/http"
	//"github.com/gin-contrib/cors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	//"time"
	//"github.com/dgrijalva/jwt-go"
	"crypto/md5"
	"encoding/hex"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func MD5(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		//fmt.Println("Cors, method: ", method, c.Request.Header)

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			fmt.Println("is options")
			//c.AbortWithStatus(http.StatusNoContent)
			c.JSON(http.StatusOK, "Options Request!")
		}
		// 处理请求
		c.Next()
	}
}

type User struct {
	ID          int    `gorm:"primary_key"`
	Name        string `gorm:"not_null"`
	Email       string `gorm:"not_null"`
	Password    string `gorm:"not_null"`
	PhoneNumber string `gorm:"not_null"`
	Address     string `gorm:"not_null"`
	Photo       string `gorm:"not_null"`
	IsAdmin     bool   `gorm:"not_null"`
}

type Appointment struct {
	ID     int    `gorm:"primary_key"`
	UserID int    `gorm:"not_null"`
	Time   string `gorm:"not_null"`
}

func sendAuthFailResponse(context *gin.Context) {
	context.JSON(200, gin.H{
		"code": 0,
		"msg":  "This user is not administrator!",
	})
}

func main() {

	db := connectDatabase()

	if db == nil {
		fmt.Println("database not connected")
	} else {
		if !db.Migrator().HasTable(User{}) {
			db.Migrator().CreateTable(User{})
		} else {
			fmt.Println("has user table")
		}

		if !db.Migrator().HasTable(Appointment{}) {
			db.Migrator().CreateTable(Appointment{})
		} else {
			fmt.Println("has appointment table")
		}
	}

	router := gin.Default()

	router.StaticFS("/uploads", http.Dir("./uploads"))

	router.Use(Cors())

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "home"})
	})

	router.GET("/users", func(c *gin.Context) {
		var users []User
		db.Where("is_admin != ?", true).Find(&users)
		fmt.Println("users", users)

		c.JSON(200, gin.H{
			"code":  1,
			"msg":   "success",
			"users": users,
		})
	})

	router.POST("/upload", func(c *gin.Context) {

		file, err := c.FormFile("file")

		fmt.Println("get upload>>>>>>>>", "file name:", file.Filename)

		if err != nil {
			fmt.Println("error:" + err.Error())
			return
		} else {
			fmt.Println("no error")
		}

		path := "./uploads/" + file.Filename
		err = c.SaveUploadedFile(file, path)

		if err != nil {
			fmt.Println("save error:" + err.Error())
		}

		//c.String(http.StatusOK, "%s uploaded!", file.Filename)
		c.JSON(200, gin.H{"path": "/uploads/" + file.Filename})
	})

	router.POST("/register", func(c *gin.Context) {
		if isUserExist(c.PostForm("email"), db) {
			c.JSON(200, gin.H{
				"code": 0,
				"msg":  "User already exist",
			})

			return
		}

		user := &User{
			Name:        c.PostForm("name"),
			Email:       c.PostForm("email"),
			Password:    MD5(c.PostForm("password")),
			Address:     c.PostForm("address"),
			Photo:       c.PostForm("photo"),
			PhoneNumber: c.PostForm("phone"),
			IsAdmin:     false,
		}

		db.Create(user)

		var addedUser User

		err := db.Where("email = ?", c.PostForm("email")).Find(&addedUser).Error

		if err != nil {
			fmt.Println(err)
		} else {
			appointment := &Appointment{
				UserID: addedUser.ID,
				Time:   c.PostForm("time"),
			}

			db.Create(appointment)
		}

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "success",
		})
	})

	router.POST("/appoint", func(c *gin.Context) {
		if !isPassordOk(c.PostForm("email"), c.PostForm("password"), db) {
			c.JSON(200, gin.H{
				"code": 0,
				"msg":  "Email or password error",
			})

			return
		}

		var user User

		err := db.Where("email = ?", c.PostForm("email")).Find(&user).Error

		if err != nil {
			fmt.Println(err)
			c.JSON(200, gin.H{
				"code": 0,
				"msg":  "Email or password error",
			})

			return
		} else {
			appointment := &Appointment{
				UserID: user.ID,
				Time:   c.PostForm("time"),
			}

			db.Create(appointment)
		}

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "success",
		})
	})

	router.POST("/signIn", func(c *gin.Context) {
		if !isPassordOk(c.PostForm("email"), c.PostForm("password"), db) {
			sendAuthFailResponse(c)
			return
		}

		var user User

		err := db.Where("email = ?", c.PostForm("email")).Find(&user).Error

		if err != nil {
			fmt.Println(err)
			sendAuthFailResponse(c)
			return
		} else {
			fmt.Println(user)

			if !user.IsAdmin {
				sendAuthFailResponse(c)
				return
			}
		}

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "success",
		})
	})

	//router.Use(cors.Default())
	/*router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "POST"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))*/

	config := cors.Config{
		AllowOrigins: []string{"*"},
	}

	fmt.Println(config)

	router.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func isUserExist(email string, db *gorm.DB) bool {
	var user User
	err := db.Where("email = ?", email).Find(&user).Error

	if err != nil {
		fmt.Println(err)
		return false
	} else {
		fmt.Println("exist", user.ID)
		return user.ID != 0
	}
}

func connectDatabase() *gorm.DB {
	dsn := "root:@(localhost:3306)/doctor_appointment_demo?charset=utf8&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println(err)
		return nil
	} else {
		fmt.Println("connection succedssed")
		return db
	}
}

func isPassordOk(email string, password string, db *gorm.DB) bool {
	var user User
	err := db.Where("email = ? and password = ?", email, MD5(password)).Find(&user).Error

	fmt.Println("data:", email, password)

	if err != nil {
		fmt.Println(err)
		return false
	} else {
		fmt.Println("exist", user.ID)
		return user.ID != 0
	}
}
