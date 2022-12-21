package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
        "bcrypt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)
type User struct {
	ID       int     `json:"id"`
	FullName string  `json:"fullname"`
	Age      int     `json:"age"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
}

type Movie struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Cover       string `json:"cover"`
	UserID      int    `json:"userid"`
}

type Rating struct {
	ID     int     `json:"id"`
	Movie  int     `json:"movie"`
	User   int     `json:"user"`
	Rating int     `json:"rating"`
	Review string  `json:"review"`
}

type Result interface {
    LastInsertId() (int64, error)
    RowsAffected() (int64, error)
}

func GenerateToken(x string) (string,error)  {
    hash, err := bcrypt.GenerateFromPassword([]byte(x), bcrypt.DefaultCost)
    if err != nil {
        log.Fatal(err)
    }
    
    return string(hash) ,err

}
func main() {
  
  db,err :=sql.Open("mysql", "mht67983:2txLIauE19eN2Ra5@/bludb") //I used my IBM data base since its the latest thing I worked with 
  if err !=nil {
    log.Fatal(err)
  }
  
  defer db.Close()
 // Initialize the Gin web framework
  r := gin.Default()
 // api routes 
  r.POST("/register",func(c *gin.Context){
    
    var user User 
    if err :=r.Bind(&user);err !=nil{
      r.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
    }
    result, err := db.Exec("INSERT INTO users (full_name, age, email, password) VALUES (?, ?, ?, ?) RETURNING id",user.FullName, user.Age, user.Email, user.Password)
    
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
    //I dont know why this  is not working although I researched and found that I must include RETURNING id in the above sql query but no bueno :(
    user.ID, err = result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
  })

  r.POST("/login",func(c *gin.Context){
    var user User 

    if err:=c.Bind(&user);err !=nil{
      r.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
    }

  var dbUser User
		err := db.QueryRow("SELECT id, full_name, age, email, password FROM users WHERE email = ? AND password = ? ",  
     user.Email, user.Password).Scan(&dbUser.ID, &dbUser.FullName, &dbUser.Age,   
     &dbUser.Email,&dbUser.Password)
    
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email or password is incorrect"})
			return
		}
    token,err := GenerateToken(dbUser.Password)
    c.JSON(http.StatusOK,gin.H{"token":token})
  })
  r.POST("/add-movie",func(c *gin.Context){
    var movie Movie

    if err:=c.Bind(&movie);err !=nil{
      c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
    }

    result,err := db.Exec("INSERT INTO movies (name, description, date, cover, user_id) VALUES (?, ?, ?, ?, ?)",     
       movie.Name, movie.Description, movie.Date, movie.Cover, movie.UserID)
    
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
    
    movie.ID, err = result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
    c.JSON(http.StatusOK,movie)
    
  })

  r.PUT("/edit-movie",func(c *gin.Context){

    var movie Movie

    if err:=c.Bind(&movie);err !=nil{
       c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      return 
    }
    movieID := c.Param("id")
    
    _, err = db.Exec("UPDATE movies SET name = ?, description = ?, date = ?, cover = ? WHERE id = ?", movie.Name,             movie.Description, movie.Date, movie.Cover, movieID)
    
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
    r.JSON(http.StatusOK)
  })

  r.DELETE("/delete-movie/:id", func(c *gin.Context) {

     var movie Movie

    if err:=c.Bind(&movie);err !=nil{
       c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      return 
    }
    
		movieID := c.Param("id")

		_, err = db.Exec("DELETE FROM movies WHERE id = ?", movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}		
    c.Status(http.StatusOK)
	})

  r.GET("/get-movie-info/:id", func(c *gin.Context) {
		
		movieID := c.Param("id")

		var movie Movie
		err := db.QueryRow("SELECT id, name, description, date, cover FROM movies WHERE id = ?", movieID).Scan(&movie.ID, &movie.Name, &movie.Description, &movie.Date, &movie.Cover)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		rows, err := db.Query("SELECT rating FROM ratings WHERE movie = ?", movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var ratings []int
		for rows.Next() {
			var rating int
			if err := rows.Scan(&rating); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			ratings = append(ratings, rating)
		}

		var sum int
		for _, rating := range ratings {
			sum += rating
		}
		averageRating := sum / len(ratings)

		c.JSON(http.StatusOK, gin.H{
			"id":           movie.ID,
			"name":         movie.Name,
			"description":  movie.Description,
			"date":         movie.Date,
			"cover":        movie.Cover,
			"averageRating": averageRating,
		})
	})

	r.GET("/list-movies", func(c *gin.Context) {
		
		rows, err := db.Query("SELECT id, name, description, date, cover FROM movies")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var movies []Movie
		for rows.Next() {
			var movie Movie
			if err := rows.Scan(&movie.ID, &movie.Name, &movie.Description, &movie.Date, &movie.Cover); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			movies = append(movies, movie)
		}

		
		sortBy := c.Query("sortBy")
		if sortBy == "date" {
			
			sort.Slice(movies, func(i, j int) bool {
				return movies[i].Date < movies[j].Date
			})
		} else if sortBy == "rating" {
			
			sort.Slice(movies, func(i, j int) bool {
				return movies[i].AverageRating > movies[j].AverageRating
			})
		}

		c.JSON(http.StatusOK, movies)
	})

	r.POST("/add-movie-to-watched-list", func(c *gin.Context) {
		
		var movie Movie
		if err := c.Bind(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		_, err = db.Exec("INSERT INTO watched (user, movie) VALUES (?, ?)", userID, movie.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	r.POST("/rate-and-review-movie", func(c *gin.Context) {
		
		var rating Rating
		if err := c.Bind(&rating); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the authentication token so i can know what user is rating and reviewing the movie
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM watched WHERE user = ? AND movie = ?", userID, rating.MovieID).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You must have watched the movie before rating it"})
			return
		}

		_, err = db.Exec("UPDATE ratings SET rating = ?, review = ? WHERE user = ? AND movie = ?", rating.Rating, rating.Review, userID, rating.MovieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	r.Run()
}
