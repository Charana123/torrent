System.config({
    paths: {
        'npm:': 'https://unpkg.com/'
    },
    map: {
        'video.js': 'npm:video.js/dist/video.min.js',
        'global/window': 'npm:global/window.js',
        'global/document': 'npm:global/document.js',
    }
});
