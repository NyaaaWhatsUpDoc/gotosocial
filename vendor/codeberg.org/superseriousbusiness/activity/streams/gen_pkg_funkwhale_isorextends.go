// Code generated by astool. DO NOT EDIT.

package streams

import (
	typealbum "codeberg.org/superseriousbusiness/activity/streams/impl/funkwhale/type_album"
	typeartist "codeberg.org/superseriousbusiness/activity/streams/impl/funkwhale/type_artist"
	typelibrary "codeberg.org/superseriousbusiness/activity/streams/impl/funkwhale/type_library"
	typetrack "codeberg.org/superseriousbusiness/activity/streams/impl/funkwhale/type_track"
	vocab "codeberg.org/superseriousbusiness/activity/streams/vocab"
)

// IsOrExtendsFunkwhaleAlbum returns true if the other provided type is the Album
// type or extends from the Album type.
func IsOrExtendsFunkwhaleAlbum(other vocab.Type) bool {
	return typealbum.IsOrExtendsAlbum(other)
}

// IsOrExtendsFunkwhaleArtist returns true if the other provided type is the
// Artist type or extends from the Artist type.
func IsOrExtendsFunkwhaleArtist(other vocab.Type) bool {
	return typeartist.IsOrExtendsArtist(other)
}

// IsOrExtendsFunkwhaleLibrary returns true if the other provided type is the
// Library type or extends from the Library type.
func IsOrExtendsFunkwhaleLibrary(other vocab.Type) bool {
	return typelibrary.IsOrExtendsLibrary(other)
}

// IsOrExtendsFunkwhaleTrack returns true if the other provided type is the Track
// type or extends from the Track type.
func IsOrExtendsFunkwhaleTrack(other vocab.Type) bool {
	return typetrack.IsOrExtendsTrack(other)
}
