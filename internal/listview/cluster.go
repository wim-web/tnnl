package listview

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func SelectClusterView(c ecsiface) (string, bool, error) {
	input := &ecs.ListClustersInput{}
	res, err := c.ListClusters(context.Background(), input)

	if err != nil {
		return "", false, err
	}

	clusterNames := []string{}

	for _, arn := range res.ClusterArns {
		v := strings.Split(arn, "/")
		clusterNames = append(clusterNames, v[1])
	}

	clusterName, quit, err := RenderList("Select a cluster", clusterNames)

	if quit {
		return "", true, nil
	}

	if err != nil {
		return "", false, err
	}

	return clusterName, false, nil
}
