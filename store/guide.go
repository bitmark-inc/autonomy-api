package store

import "github.com/bitmark-inc/autonomy-api/schema"

type Guide interface {
	NearbyTestCenter(loc schema.Location, limit int)
}
