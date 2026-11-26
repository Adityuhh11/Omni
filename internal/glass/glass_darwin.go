package glass

// shape does nothing on a Mac: the Glass theme's blur there is the launcher's Liquid Glass (internal/panel) and the card windows' own
// (internal/settingsui), which the page does not place. The page never calls Shape on a Mac; this keeps the shared service building.
func shape(uintptr, []Rect, bool) (Glass, error) { return Glass{Look: Blur}, nil }

// forget has nothing to do on a Mac.
func forget(uintptr) {}
