package app

// pagesKept is how many pages keep what they loaded: those just left
// behind, which going back shows again at once. A playlist of a thousand
// songs takes half a megabyte on its own, and a session walks through many
// of them.
const pagesKept = 12

// keepsPage reports whether the page of r loads something of its own into
// the app: the pages that show what the home page or the charts do don't.
func keepsPage(r route) bool {
	switch r.page {
	case pagePlaylist, pageSearch, pageArtist, pageAlbum, pageUser, pageRecord, pagePodcast:
		return true
	}
	return false
}

// notePage takes r as the page shown, and drops what the pages before it
// loaded once more than pagesKept of them are kept, so that a session
// through many pages doesn't hold them all.
func (a *app) notePage(r route) {
	if !keepsPage(r) {
		return
	}
	for i := range a.pagesUsed {
		if a.pagesUsed[i] == r {
			a.pagesUsed = append(a.pagesUsed[:i], a.pagesUsed[i+1:]...)
			break
		}
	}
	a.pagesUsed = append(a.pagesUsed, r)
	for len(a.pagesUsed) > pagesKept {
		old := a.pagesUsed[0]
		// Its load is on its way: the page would be dropped before it
		// lands, and what landed would be kept with no page to belong to.
		if a.pageLoading(old) {
			break
		}
		a.pagesUsed = append(a.pagesUsed[:0], a.pagesUsed[1:]...)
		a.dropPage(old)
	}
}

// dropPage drops what the page of r loaded: its songs, its albums, or a
// search's results.
func (a *app) dropPage(r route) {
	switch r.page {
	case pagePlaylist:
		delete(a.playlists, r.id)
		delete(a.plErr, r.id)
	case pageArtist:
		delete(a.artists, r.id)
	case pageAlbum:
		delete(a.albums, r.id)
	case pageUser:
		delete(a.users, r.id)
	case pageRecord:
		delete(a.records, r.id)
	case pagePodcast:
		delete(a.podcasts, r.id)
	case pageSearch:
		delete(a.searches, r)
	}
}

// pageLoading reports whether the page of r waits on NetEase still.
func (a *app) pageLoading(r route) bool {
	switch r.page {
	case pagePlaylist:
		return a.loadingPl[r.id]
	case pageArtist:
		if d := a.artists[r.id]; d != nil {
			return d.loading || d.latestLoading
		}
	case pageAlbum:
		if d := a.albums[r.id]; d != nil {
			return d.loading
		}
	case pageUser:
		if d := a.users[r.id]; d != nil {
			return d.loading > 0 || d.subsLoading > 0
		}
	case pageRecord:
		if d := a.records[r.id]; d != nil {
			return d.loading > 0
		}
	case pagePodcast:
		if d := a.podcasts[r.id]; d != nil {
			return d.loading || d.page.loading
		}
	case pageSearch:
		if d := a.searches[r]; d != nil {
			return d.page.loading
		}
	}
	return false
}
