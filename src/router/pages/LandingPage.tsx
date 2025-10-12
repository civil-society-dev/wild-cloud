import { useNavigate } from 'react-router';
import { useInstanceContext } from '../../hooks/useInstanceContext';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Server } from 'lucide-react';

export function LandingPage() {
  const navigate = useNavigate();
  const { currentInstance } = useInstanceContext();

  // For now, we'll use a default instance
  // In the future, this will show an instance selector
  const handleSelectInstance = () => {
    const instanceId = currentInstance || 'default';
    navigate(`/instances/${instanceId}/dashboard`);
  };

  return (
    <div className="flex items-center justify-center min-h-screen">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Wild Cloud</CardTitle>
          <CardDescription>
            Select an instance to manage your cloud infrastructure
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button
            onClick={handleSelectInstance}
            className="w-full"
            size="lg"
          >
            <Server className="mr-2 h-5 w-5" />
            {currentInstance ? `Continue to ${currentInstance}` : 'Go to Default Instance'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
