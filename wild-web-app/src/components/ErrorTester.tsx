import { useState } from 'react';
import { Button } from './ui/button';
import { Card, CardHeader, CardTitle, CardContent } from './ui/card';
import { AlertTriangle } from 'lucide-react';

// Component that can trigger errors for testing
export const ErrorTester = () => {
  const [shouldThrow, setShouldThrow] = useState(false);

  if (shouldThrow) {
    throw new Error('Test error: This is a simulated component crash for testing the error boundary.');
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <AlertTriangle className="h-5 w-5 text-yellow-600" />
          Error Boundary Tester
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          This component can be used to test the error boundary functionality in development.
        </p>
        
        <div className="p-3 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded-md">
          <p className="text-sm text-yellow-800 dark:text-yellow-200">
            ⚠️ Warning: Clicking the button below will intentionally crash this component to test error handling.
          </p>
        </div>
        
        <Button 
          onClick={() => setShouldThrow(true)} 
          variant="destructive"
          size="sm"
        >
          Trigger Error
        </Button>
        
        <div className="text-xs text-muted-foreground">
          Development tool - remove from production builds
        </div>
      </CardContent>
    </Card>
  );
};