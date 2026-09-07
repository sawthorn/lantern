import { prevTrack, nextTrack } from "./utils.js";

const audio = document.getElementById('player');
const btnPlay = document.getElementById('btn-play');
const iconPlay = document.getElementById('icon-play');
const iconPause = document.getElementById('icon-pause');
const seekInput = document.getElementById('seek-input');
const seekFill = document.getElementById('seek-fill');
const seekThumb = document.getElementById('seek-thumb');
const currentTimeEl = document.getElementById('current-time');
const durationEl = document.getElementById('duration');
const volInput = document.getElementById('vol-input');
const volDisplay = document.getElementById('vol-display');
const btnMute = document.getElementById('btn-mute');
const iconVol = document.getElementById('icon-vol');
const iconMute = document.getElementById('icon-mute');
const btnLoop = document.getElementById('btn-loop');
const btnShuffle = document.getElementById('btn-shuffle');
const btnPrev = document.getElementById('btn-prev');
const btnNext = document.getElementById('btn-next');

let looping = false;
let prevMuted = false;

function fmt(s) {
    if (isNaN(s)) return '0:00';
    const m = Math.floor(s / 60);
    const sec = Math.floor(s % 60);
    return `${m}:${sec.toString().padStart(2, '0')}`;
}

function setPlaying(isPlaying) {
    iconPlay.style.display = isPlaying ? 'none' : 'block';
    iconPause.style.display = isPlaying ? 'block' : 'none';
}

function updateSeek() {
    if (!audio.duration) return;
    const pct = (audio.currentTime / audio.duration) * 100;
    seekFill.style.width = pct + '%';
    seekInput.value = pct;
    seekThumb.style.left = pct + '%';
    currentTimeEl.textContent = fmt(audio.currentTime);
}

// play/pause
btnPlay.addEventListener('click', () => {
    if (!audio.src) return;
    audio.paused ? audio.play() : audio.pause();
});

audio.addEventListener('play', () => setPlaying(true));
audio.addEventListener('pause', () => setPlaying(false));
audio.addEventListener('ended', () => setPlaying(false));
audio.addEventListener('timeupdate', updateSeek);
audio.addEventListener('loadedmetadata', () => {
    durationEl.textContent = fmt(audio.duration);
});

// seek
seekInput.addEventListener('input', () => {
    if (!audio.duration) return;
    const t = (seekInput.value / 100) * audio.duration;
    audio.currentTime = t;
    seekFill.style.width = seekInput.value + '%';
    seekThumb.style.left = seekInput.value + '%';
});

// prev = restart
btnPrev.addEventListener('click', () => {
    if (audio.currentTime <= 3) {
        prevTrack()
    }
    audio.currentTime = 0;
    if (audio.paused && audio.src) audio.play();
});

// next = skip to end
btnNext.addEventListener('click', () => {
    nextTrack()
});

// volume
volInput.addEventListener('input', () => {
    audio.volume = volInput.value;
    volDisplay.textContent = Math.round(volInput.value * 100);
    if (audio.volume === 0) {
        iconVol.style.display = 'none';
        iconMute.style.display = 'block';
    } else {
        iconVol.style.display = 'block';
        iconMute.style.display = 'none';
    }
});
audio.volume = 0.8;

// mute toggle
btnMute.addEventListener('click', () => {
    audio.muted = !audio.muted;
    iconVol.style.display = audio.muted ? 'none' : 'block';
    iconMute.style.display = audio.muted ? 'block' : 'none';
});

// // loop
// btnLoop.addEventListener('click', () => {
//     looping = !looping;
//     audio.loop = looping;
//     btnLoop.classList.toggle('active', looping);
// });
//
// // shuffle (visual only for single file)
// btnShuffle.addEventListener('click', () => {
//     btnShuffle.classList.toggle('active');
// });
