package dock

import "github.com/gin-gonic/gin"

func ServeHttp(fn func(r *gin.Engine), addr ...string) error {
	r := gin.Default()
	fn(r)
	// listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	if err := r.Run(addr...); err != nil {
		return err
	}
	return nil
}
