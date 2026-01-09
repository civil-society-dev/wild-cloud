import { useNavigate } from 'react-router';
import { useInstances } from '../../hooks/useInstances';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Server, CloudLightning, Loader2, Plus } from 'lucide-react';
import { useState } from 'react';

export function LandingPage() {
  const navigate = useNavigate();
  const { instances, isLoading, error, createInstance, isCreating } = useInstances();
  const [newInstanceName, setNewInstanceName] = useState('');

  const handleSelectInstance = (instanceName: string) => {
    navigate(`/instances/${instanceName}/dashboard`);
  };

  const handleCreateInstance = () => {
    // TODO: Show a modal/dialog to collect instance name and configuration
    const instanceName = prompt('Enter instance name:');
    if (instanceName) {
      createInstance({ name: instanceName });
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-background">
      <div className="container max-w-2xl px-4">
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <CloudLightning className="h-12 w-12 text-primary" />
            <h1 className="text-4xl font-bold">Wild Cloud</h1>
          </div>
          <p className="text-lg text-muted-foreground">
            Manage your cloud infrastructure with ease
          </p>
        </div>

        <Card>
          <CardHeader className="text-center">
            <CardTitle className="text-xl">Clouds</CardTitle>
            <CardDescription>
              Select a Wild Cloud instance to manage
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {isLoading && (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            )}
            {error && (
              <p className="text-sm text-red-600 text-center py-4">
                Failed to load instances
              </p>
            )}
            {!isLoading && !error && instances.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-4">
                No clouds available. Create one to get started.
              </p>
            )}
            {!isLoading && !error && instances.map((instanceName: string) => (
              <Button
                key={instanceName}
                onClick={() => handleSelectInstance(instanceName)}
                className="w-full"
                size="lg"
                variant="outline"
              >
                <Server className="mr-2 h-5 w-5" />
                {instanceName}
              </Button>
            ))}
          </CardContent>
          <CardFooter>
            <Button
              onClick={handleCreateInstance}
              className="w-full"
              variant="default"
              disabled={isCreating}
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Plus className="mr-2 h-5 w-5" />
                  Create New Cloud
                </>
              )}
            </Button>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
