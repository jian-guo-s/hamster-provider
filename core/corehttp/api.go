package corehttp

import (
	"fmt"
	"github.com/hamster-shared/hamster-provider/core/context"
)

func StartApi(ctx *context.CoreContext) error {
	r := NewMyServer(ctx)
	// router
	v1 := r.Group("/api/v1")
	{
		// container
		container := v1.Group("/container")
		{
			container.GET("/start", startContainer)
			container.GET("/delete", deleteContainer)
		}

		pk := v1.Group("/pk")
		// publick key
		{
			pk.POST("/grantKey", grantKey)
			pk.POST("/deleteKey", deleteKey)
			pk.POST("/queryKey", queryKey)
		}

		p2p := v1.Group("/p2p")
		// p2p
		{
			p2p.POST("/listen", listenP2p)
			p2p.POST("/forward", forwardP2p)
			p2p.POST("/ls", lsP2p)
			p2p.POST("/close", closeP2p)
			p2p.POST("/check", checkP2p)
		}
		vm := v1.Group("/vm")
		{
			vm.POST("/create", createVm)
		}
		resource := v1.Group("/resource")
		{
			resource.POST("/modify-price", modifyPrice)
			resource.POST("/add-duration", addDuration)
		}
	}
	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	port := ctx.GetConfig().ApiPort
	return r.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
