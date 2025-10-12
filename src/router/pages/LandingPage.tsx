import { useNavigate, Link } from 'react-router';
import { useInstanceContext } from '../../hooks/useInstanceContext';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Server, Usb, HardDrive, CloudLightning } from 'lucide-react';

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
    <div className="flex items-center justify-center min-h-screen bg-background">
      <div className="container max-w-4xl px-4">
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <CloudLightning className="h-12 w-12 text-primary" />
            <h1 className="text-4xl font-bold">Wild Cloud</h1>
          </div>
          <p className="text-lg text-muted-foreground">
            Manage your cloud infrastructure with ease
          </p>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader className="text-center">
              <CardTitle className="text-xl">Cloud Instance</CardTitle>
              <CardDescription>
                Manage your Wild Cloud instance
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

          <Card>
            <CardHeader className="text-center">
              <CardTitle className="text-xl">Boot Assets</CardTitle>
              <CardDescription>
                Download Talos installation media
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Link to="/iso" className="block">
                <Button
                  variant="outline"
                  className="w-full"
                  size="lg"
                >
                  <Usb className="mr-2 h-5 w-5" />
                  ISO / USB Boot
                </Button>
              </Link>
              <Link to="/pxe" className="block">
                <Button
                  variant="outline"
                  className="w-full"
                  size="lg"
                >
                  <HardDrive className="mr-2 h-5 w-5" />
                  PXE Network Boot
                </Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
