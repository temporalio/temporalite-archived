package temporalite

import (
	"context"
	sql2 "database/sql"
	"fmt"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/config"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/serialization"
	"go.temporal.io/server/common/persistence/sql"
	"go.temporal.io/server/common/persistence/sql/sqlplugin"
	"go.temporal.io/server/common/resolver"
)

type searchAttributeHelper struct {
}

// getClusterMeta will return the de-serialized cluster metadata from the DB if it's present. Otherwise, it will
// initialize one in-memory
func getClusterMeta(clusterConfig *cluster.Config, cfg *config.SQL) (*persistence.ClusterMetadata, error) {
	db, err := sql.NewSQLDB(sqlplugin.DbKindUnknown, cfg, resolver.NewNoopResolver())
	if err != nil {
		return nil, fmt.Errorf("unable to create SQLite admin DB: %w", err)
	}
	defer func() { _ = db.Close() }()
	row, err := db.GetClusterMetadata(context.Background(), &sqlplugin.ClusterMetadataFilter{
		ClusterName: clusterConfig.CurrentClusterName,
	})
	if err != nil && err != sql2.ErrNoRows {
		return nil, err
	}
	if row == nil {
		row = &sqlplugin.ClusterMetadataRow{
			ClusterName:  clusterConfig.CurrentClusterName,
			DataEncoding: enums.ENCODING_TYPE_PROTO3.String(),
			Version:      0,
		}
	}
	blob := p.NewDataBlob(row.Data, row.DataEncoding)
	clusterMeta, err := serialization.NewSerializer().DeserializeClusterMetadata(blob)
	if err != nil {
		return nil, err
	}
	var initialFailover int64 = 1
	var RPCAddress = ""
	if cc, ok := clusterConfig.ClusterInformation[clusterConfig.CurrentClusterName]; ok {
		initialFailover = cc.InitialFailoverVersion
		RPCAddress = cc.RPCAddress
	}
	if clusterMeta == nil {
		return &persistence.ClusterMetadata{
			HistoryShardCount:        int32(1),
			ClusterName:              clusterConfig.CurrentClusterName,
			FailoverVersionIncrement: 10,
			InitialFailoverVersion:   initialFailover,
			IsGlobalNamespaceEnabled: clusterConfig.EnableGlobalNamespace,
			ClusterAddress:           RPCAddress,
		}, nil
	}
	return clusterMeta, nil
}

func AddSearchAttributes(clusterConfig *cluster.Config, cfg *config.SQL, searchAttributes map[string]enums.IndexedValueType) error {
	db, err := sql.NewSQLDB(sqlplugin.DbKindUnknown, cfg, resolver.NewNoopResolver())
	if err != nil {
		return fmt.Errorf("unable to create SQLite admin DB: %w", err)
	}
	defer func() { _ = db.Close() }()

	clusterMeta, err := getClusterMeta(clusterConfig, cfg)
	if err != nil {
		return err
	}
	if clusterMeta.IndexSearchAttributes == nil {
		clusterMeta.IndexSearchAttributes = map[string]*persistence.IndexSearchAttributes{}
	}
	if clusterMeta.IndexSearchAttributes[""] == nil {
		clusterMeta.IndexSearchAttributes[""] = &persistence.IndexSearchAttributes{
			CustomSearchAttributes: map[string]enums.IndexedValueType{},
		}
	}

	for key, value := range searchAttributes {
		clusterMeta.IndexSearchAttributes[""].CustomSearchAttributes[key] = value
	}
	serializer := serialization.NewSerializer()

	dataBlob, err := serializer.SerializeClusterMetadata(clusterMeta, enums.ENCODING_TYPE_PROTO3)
	if err != nil {
		return err
	}
	var metaVersion int64
	metaRow, err := db.WriteLockGetClusterMetadataV1(context.Background())
	if err != nil {
		if err != sql2.ErrNoRows {
			return err
		}
	} else {
		metaVersion = metaRow.Version
	}
	_, err = db.SaveClusterMetadata(context.Background(), &sqlplugin.ClusterMetadataRow{
		ClusterName:  clusterConfig.CurrentClusterName,
		Data:         dataBlob.Data,
		DataEncoding: enums.ENCODING_TYPE_PROTO3.String(),
		Version:      metaVersion,
	})
	return err
}
