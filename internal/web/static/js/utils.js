export {updatePlayer, className, prevTrack, nextTrack};

let currentTrackElem = null;
let player;
let playerTrackTitle;
let playerTrackArtist;
let playerTrackArt;

document.addEventListener('DOMContentLoaded', () => {
    player = document.getElementById("player")
    player.addEventListener('ended', nextTrack)
    playerTrackTitle = document.getElementById('track-title');
    playerTrackArtist = document.getElementById('track-artist');
    playerTrackArt = document.getElementById('track-art');
});

async function updatePlayer(track) {
    if (currentTrackElem) {
        currentTrackElem.classList.remove("active-track")
    }

    currentTrackElem = track
    currentTrackElem.classList.add("active-track")

    player.src = `/api/stream/${track.dataset.trackId}`
    playerTrackTitle.textContent = track.dataset.title
    playerTrackArtist.textContent = track.dataset.artist
    playerTrackArt.replaceChildren(
        getTrackCover(track.dataset.albumId)
    )

    player.play()
}

async function prevTrack() {
    if (!currentTrackElem) return;

    const prevTrack = currentTrackElem.previousElementSibling;
    if (prevTrack) {
        updatePlayer(prevTrack)
    }
}

async function nextTrack() {
    if (!currentTrackElem) return;

    const nextTrack = currentTrackElem.nextElementSibling;
    if (nextTrack) {
        updatePlayer(nextTrack)
    }
}


function getTrackCover(id) {
    const img = document.createElement("img")
    img.src = `/api/cover/${id}`
    return img
}

function className(elem, cls) {
    const classes = cls.split(" ")
    for (const cl of classes) {
        elem.classList.add(cl)
    }
}
