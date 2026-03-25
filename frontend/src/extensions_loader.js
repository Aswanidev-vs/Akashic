import { GetExtensions } from '../wailsjs/go/main/App.js';

(async () => {
  try {
    const files = await GetExtensions();
    files.forEach((file) => {
      if (file.endsWith('.js')) {
        const script = document.createElement('script');
        script.type = 'module';
        script.src = `./extensions/${file}`;
        document.body.appendChild(script);
      } else if (file.endsWith('.css')) {
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = `./extensions/${file}`;
        document.head.appendChild(link);
      }
    });
  } catch (e) {
    console.error('Failed to load extensions:', e);
  }
})();
