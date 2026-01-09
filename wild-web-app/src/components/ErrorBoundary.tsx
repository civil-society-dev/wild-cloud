import React, { Component as ReactComponent, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';
import { Button } from './ui/button';
import { Card, CardHeader, CardTitle, CardContent } from './ui/card';

interface Props {
  children?: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

export class ErrorBoundary extends ReactComponent<Props, State> {
  public state: State = {
    hasError: false,
  };

  public static getDerivedStateFromError(error: Error): State {
    // Update state so the next render will show the fallback UI
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    
    this.setState({
      error,
      errorInfo,
    });

    // Call optional error handler
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  private handleReset = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined });
  };

  private handleReload = () => {
    window.location.reload();
  };

  public render() {
    if (this.state.hasError) {
      // If a custom fallback is provided, use it
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return <ErrorFallback 
        error={this.state.error} 
        errorInfo={this.state.errorInfo}
        onReset={this.handleReset}
        onReload={this.handleReload}
      />;
    }

    return this.props.children;
  }
}

interface ErrorFallbackProps {
  error?: Error;
  errorInfo?: ErrorInfo;
  onReset: () => void;
  onReload: () => void;
}

export const ErrorFallback: React.FC<ErrorFallbackProps> = ({ 
  error, 
  errorInfo, 
  onReset, 
  onReload 
}) => {
  const isDev = process.env.NODE_ENV === 'development';

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <Card className="w-full max-w-2xl">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg">
              <AlertTriangle className="h-6 w-6 text-red-600 dark:text-red-400" />
            </div>
            <div>
              <CardTitle className="text-red-800 dark:text-red-200">
                Something went wrong
              </CardTitle>
              <p className="text-sm text-muted-foreground mt-1">
                The application encountered an unexpected error
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Don't worry, your data is safe. You can try the following options:
            </p>
            <div className="flex gap-2">
              <Button onClick={onReset} variant="outline" size="sm">
                <RefreshCw className="h-4 w-4 mr-2" />
                Try Again
              </Button>
              <Button onClick={onReload} variant="outline" size="sm">
                <Home className="h-4 w-4 mr-2" />
                Reload Page
              </Button>
            </div>
          </div>

          {isDev && error && (
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-red-800 dark:text-red-200">
                Error Details (Development Mode)
              </h4>
              
              <div className="space-y-2">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Error Message:</p>
                  <p className="text-xs bg-red-50 p-2 rounded border font-mono">
                    {error.message}
                  </p>
                </div>
                
                {error.stack && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground">Stack Trace:</p>
                    <pre className="text-xs p-2 rounded border overflow-auto max-h-40 font-mono">
                      {error.stack}
                    </pre>
                  </div>
                )}
                
                {errorInfo?.componentStack && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground">Component Stack:</p>
                    <pre className="text-xs p-2 rounded border overflow-auto max-h-40 font-mono">
                      {errorInfo.componentStack}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          )}
          
          {!isDev && (
            <div className="p-3 bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-md">
              <p className="text-sm text-blue-800 dark:text-blue-200">
                If this problem persists, please contact support with details about what you were doing when the error occurred.
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};