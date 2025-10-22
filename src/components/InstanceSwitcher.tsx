import { useState } from 'react';
import { useNavigate, useLocation, useParams } from 'react-router';
import { Plus } from 'lucide-react';
import { useInstances } from '../hooks/useInstances';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectSeparator,
} from './ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';

const ADD_INSTANCE_VALUE = '__add_new__';

export function InstanceSwitcher() {
  const navigate = useNavigate();
  const location = useLocation();
  const { instanceId } = useParams<{ instanceId: string }>();
  const { instances, isLoading, error, createInstance, isCreating } = useInstances();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [newInstanceName, setNewInstanceName] = useState('');

  const handleInstanceChange = (value: string) => {
    // Check if user selected "Add new instance"
    if (value === ADD_INSTANCE_VALUE) {
      setDialogOpen(true);
      return;
    }

    if (!instanceId) return;

    // Extract the page path after /instances/:instanceId
    const instancePrefix = `/instances/${instanceId}`;
    const pagePath = location.pathname.startsWith(instancePrefix)
      ? location.pathname.slice(instancePrefix.length)
      : '/dashboard';

    // Navigate to the same page in the new instance
    navigate(`/instances/${value}${pagePath || '/dashboard'}`);
  };

  const handleCreateInstance = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newInstanceName.trim()) return;

    createInstance(
      { name: newInstanceName.trim() },
      {
        onSuccess: () => {
          setDialogOpen(false);
          setNewInstanceName('');
          // Navigate to the new instance's dashboard
          navigate(`/instances/${newInstanceName.trim()}/dashboard`);
        },
      }
    );
  };

  // Loading state
  if (isLoading) {
    return (
      <Select disabled value="">
        <SelectTrigger className="h-8 text-sm">
          <SelectValue placeholder="Loading..." />
        </SelectTrigger>
      </Select>
    );
  }

  // Error state
  if (error) {
    return (
      <Select disabled value="">
        <SelectTrigger className="h-8 text-sm">
          <SelectValue placeholder="Error loading instances" />
        </SelectTrigger>
      </Select>
    );
  }

  // No instances state - show dialog immediately
  if (!instances || instances.length === 0) {
    return (
      <>
        <Select disabled value="">
          <SelectTrigger className="h-8 text-sm">
            <SelectValue placeholder="No instances" />
          </SelectTrigger>
        </Select>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setDialogOpen(true)}
          className="mt-2 w-full h-8 text-sm"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Instance
        </Button>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent>
            <form onSubmit={handleCreateInstance}>
              <DialogHeader>
                <DialogTitle>Create New Instance</DialogTitle>
                <DialogDescription>
                  Enter a name for your new Wild Cloud instance.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Instance Name</Label>
                  <Input
                    id="name"
                    placeholder="my-instance"
                    value={newInstanceName}
                    onChange={(e) => setNewInstanceName(e.target.value)}
                    disabled={isCreating}
                    autoFocus
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setDialogOpen(false)}
                  disabled={isCreating}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={isCreating || !newInstanceName.trim()}>
                  {isCreating ? 'Creating...' : 'Create Instance'}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </>
    );
  }

  return (
    <>
      <Select value={instanceId || ''} onValueChange={handleInstanceChange}>
        <SelectTrigger className="h-8 text-sm">
          <SelectValue placeholder="Select instance" />
        </SelectTrigger>
        <SelectContent>
          {instances.map((instance) => (
            <SelectItem key={instance} value={instance}>
              {instance}
            </SelectItem>
          ))}
          <SelectSeparator />
          <SelectItem value={ADD_INSTANCE_VALUE}>
            <div className="flex items-center">
              <Plus className="h-4 w-4 mr-2" />
              Add new instance...
            </div>
          </SelectItem>
        </SelectContent>
      </Select>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <form onSubmit={handleCreateInstance}>
            <DialogHeader>
              <DialogTitle>Create New Instance</DialogTitle>
              <DialogDescription>
                Enter a name for your new Wild Cloud instance.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Instance Name</Label>
                <Input
                  id="name"
                  placeholder="my-instance"
                  value={newInstanceName}
                  onChange={(e) => setNewInstanceName(e.target.value)}
                  disabled={isCreating}
                  autoFocus
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
                disabled={isCreating}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isCreating || !newInstanceName.trim()}>
                {isCreating ? 'Creating...' : 'Create Instance'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
