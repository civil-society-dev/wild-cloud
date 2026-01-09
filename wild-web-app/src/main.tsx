import { StrictMode } from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';
import './index.css';
import App from './App';
import { ThemeProvider } from './contexts/ThemeContext';
import { InstanceProvider } from './hooks';
import { queryClient } from './lib/queryClient';
import { ErrorBoundary } from './components/ErrorBoundary';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <InstanceProvider>
          <ThemeProvider defaultTheme="light" storageKey="wild-central-theme">
            <App />
          </ThemeProvider>
        </InstanceProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>
);