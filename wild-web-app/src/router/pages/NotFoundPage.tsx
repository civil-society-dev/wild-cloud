import { Link } from 'react-router';
import { AlertCircle, Home } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';

export function NotFoundPage() {
  return (
    <div className="flex items-center justify-center min-h-[600px]">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-4">
            <AlertCircle className="h-16 w-16 text-destructive" />
          </div>
          <CardTitle className="text-2xl">Page Not Found</CardTitle>
          <CardDescription>
            The page you're looking for doesn't exist or has been moved.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-center">
          <Link to="/">
            <Button>
              <Home className="mr-2 h-4 w-4" />
              Go to Home
            </Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
