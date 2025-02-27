package snapshotManifest

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_CreatedAtTimeParsing(t *testing.T) {
	inputJson := `
		{
			"createdAt": "2025-08-10T15:00:00Z"
		}
	`
	var snapshot *Snapshot
	err := json.Unmarshal([]byte(inputJson), &snapshot)
	assert.Nil(t, err)
	assert.Equal(t, snapshot.CreatedAt.Format(time.RFC3339), "2025-08-10T15:00:00Z")
}

func Test_UnmarshalJsonToManifest(t *testing.T) {
	inputJson := `
		{
	"metadata": {
		"version": "v1.0.0"
	},
	"snapshots": [
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T15:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227153000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T14:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227143001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T13:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227133001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T12:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227123000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T12:00:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250227120001.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T11:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227113000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T10:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227103000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T09:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227093000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T08:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227083000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T08:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250227080000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T07:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227073000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T06:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227063000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T05:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227053000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T04:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227043001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T04:00:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250227040001.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T03:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227033000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T02:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227023001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T01:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227013000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T00:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250227003000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-27T00:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250227000000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T23:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226233000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T22:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226223001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T21:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226213000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T20:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226203001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T20:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250226200000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T19:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226193001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T18:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226183000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T17:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226173000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T16:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226163000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T16:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.4.0_public_20250226160000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T15:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226153000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.4.0",
			"createdAt": "2025-02-26T14:30:15Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.4.0_public_20250226143015.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T13:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226133000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T12:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226123000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T12:00:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.3.1+6e29089_public_20250226120001.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T11:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226113000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T10:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226103000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T09:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226093000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T08:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226083001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T08:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.3.1+6e29089_public_20250226080000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T07:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226073000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T06:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226063000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T05:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226053000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T04:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226043001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T04:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.3.1+6e29089_public_20250226040000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T03:30:14Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226033014.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T02:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226023001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T01:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226013000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T00:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250226003000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-26T00:00:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_full_v2.3.1+6e29089_public_20250226000000.dump",
			"schema": "public",
			"kind": "full",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-25T23:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250225233000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-25T22:30:00Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250225223000.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-25T21:30:01Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250225213001.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		},
		{
			"sidecarVersion": "v2.3.1+6e29089",
			"createdAt": "2025-02-25T21:00:19Z",
			"chain": "mainnet",
			"url": "https://sidecar.eigenlayer.xyz/snapshots/mainnet/sidecar_mainnet_slim_v2.3.1+6e29089_public_20250225210019.dump",
			"schema": "public",
			"kind": "slim",
			"signature": ""
		}
	]
}
	`

	manifest, err := NewSnapshotManifestFromJson([]byte(inputJson))
	assert.Nil(t, err)
	assert.Equal(t, manifest.Metadata.Version, "v1.0.0")
	assert.Equal(t, len(manifest.Snapshots), 54)
}
