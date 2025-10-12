import { createBrowserRouter } from 'react-router';
import { routes } from './routes';

export const router = createBrowserRouter(routes, {
  future: {
    v7_startTransition: true,
    v7_relativeSplatPath: true,
  },
});

export { routes };
export * from './InstanceLayout';
export * from './pages/LandingPage';
export * from './pages/NotFoundPage';
