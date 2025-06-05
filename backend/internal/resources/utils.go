package resources

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// GetResourceTypeMap fetches all resource types and returns a map of their IDs to names.
func GetResourceTypeMap(app *pocketbase.PocketBase) (map[string]string, error) {
	log.Printf("DEBUG: GetResourceTypeMap called")
	resourceTypes, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
	if err != nil {
		log.Printf("ERROR: GetResourceTypeMap: Failed to fetch resource_types: %v", err)
		return nil, fmt.Errorf("failed to fetch resource_types: %w", err)
	}

	resourceMap := make(map[string]string)
	for _, rt := range resourceTypes {
		resourceMap[rt.Id] = rt.GetString("name")
	}

	log.Printf("DEBUG: GetResourceTypeMap: Successfully created resource type map with %d entries", len(resourceMap))
	return resourceMap, nil
}

// GetResourceTypeIdFromName fetches a single resource type ID by its name.
// Returns the ID or an empty string if not found, and an error if the query fails.
func GetResourceTypeIdFromName(app *pocketbase.PocketBase, name string) (string, error) {
    log.Printf("DEBUG: GetResourceTypeIdFromName called for name: %s", name)
    resourceTypeRecord, err := app.Dao().FindFirstRecordByFilter("resource_types", "name = {:name}", dbx.Params{"name": name})
    if err != nil {
        log.Printf("ERROR: GetResourceTypeIdFromName: Failed to query resource_type '%s': %v", name, err)
        return "", fmt.Errorf("failed to query resource_type '%s': %w", name, err)
    }
    if resourceTypeRecord == nil {
        log.Printf("WARN: GetResourceTypeIdFromName: Resource type '%s' not found.", name)
        return "", nil // Not found is not necessarily a hard error, could be expected by caller
    }
    return resourceTypeRecord.Id, nil
}
