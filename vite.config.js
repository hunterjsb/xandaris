import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    {
      name: 'configure-server',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/health') {
            res.statusCode = 200;
            res.end('OK');
            return;
          }
          next();
        });
      }
    }
  ],
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: [
      '15c8-2600-1700-3270-3af0-3360-821e-f3a5-4421.ngrok-free.app',
      'xandaris.vibechuck.com'
    ]
  },
  build: {
    outDir: 'web',
    rollupOptions: {
      input: {
        main: 'index.html'
      }
    }
  }
});