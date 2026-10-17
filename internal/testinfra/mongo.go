package testinfra

import (
	"context"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const MongoImage = "mongo:8.2" // mongo:8.0 does not start on Linux kernels >= 6.19 (SERVER-121912)

// StartMongo starts a single-node replica set: transactions do not work on a
// standalone mongod. URI = "mongodb://...".
func StartMongo(ctx context.Context) (*Container, error) {
	c, err := mongodb.Run(ctx, MongoImage, mongodb.WithReplicaSet("rs0"))
	if err != nil {
		return nil, err
	}
	uri, err := c.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	// The replica set advertises the container's internal address, which is not
	// reachable from the host: connect directly to the one node instead.
	sep := "?"
	if strings.Contains(uri, "?") {
		sep = "&"
	}
	return &Container{Container: c, URI: uri + sep + "directConnection=true"}, nil
}
