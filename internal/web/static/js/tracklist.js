import { updatePlayer } from "./utils.js";

const tracklist = document.getElementById("tracklist")

tracklist.addEventListener("click", (e) => {
    const track = e.target.closest(".tracklist-entry")

    if (!track) {
        return
    }

    updatePlayer(track)
})

