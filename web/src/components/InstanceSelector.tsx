import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Cloud, Plus, Check, Loader2, AlertCircle } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useInstances } from '../hooks/useInstances';

export function InstanceSelector() {
  const { currentInstance, setCurrentInstance } = useInstanceContext();
  const { instances, isLoading, error, createInstance, isCreating } = useInstances();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newInstanceName, setNewInstanceName] = useState('');

  const handleSelectInstance = (name: string) => {
    setCurrentInstance(name);
  };

  const handleCreateInstance = () => {
    if (!newInstanceName.trim()) return;
    createInstance({ name: newInstanceName.trim() });
    setShowCreateForm(false);
    setNewInstanceName('');
  };

  if (isLoading) {
    return (
      <Card className="p-4">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span className="text-sm">Loading instances...</span>
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="p-4 border-red-200 bg-red-50 dark:bg-red-950/20">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-4 w-4 text-red-500" />
          <span className="text-sm text-red-700 dark:text-red-300">
            Error loading instances: {(error as Error).message}
          </span>
        </div>
      </Card>
    );
  }

  return (
    <Card className="p-4">
      <div className="flex items-center gap-4">
        <Cloud className="h-5 w-5 text-primary" />
        <div className="flex-1">
          <label className="text-sm font-medium mb-1 block">Instance</label>
          <select
            value={currentInstance || ''}
            onChange={(e) => handleSelectInstance(e.target.value)}
            className="w-full px-3 py-2 border rounded-lg bg-background"
          >
            <option value="">Select an instance...</option>
            {instances.map((instance) => (
              <option key={instance} value={instance}>
                {instance}
              </option>
            ))}
          </select>
        </div>

        {currentInstance && (
          <Badge variant="success" className="whitespace-nowrap">
            <Check className="h-3 w-3 mr-1" />
            Active
          </Badge>
        )}

        <Button
          size="sm"
          variant="outline"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          <Plus className="h-4 w-4 mr-1" />
          New
        </Button>
      </div>

      {showCreateForm && (
        <div className="mt-4 pt-4 border-t">
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="Instance name"
              value={newInstanceName}
              onChange={(e) => setNewInstanceName(e.target.value)}
              className="flex-1 px-3 py-2 border rounded-lg"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && newInstanceName.trim()) {
                  handleCreateInstance();
                }
              }}
            />
            <Button
              size="sm"
              onClick={handleCreateInstance}
              disabled={!newInstanceName.trim() || isCreating}
            >
              {isCreating ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Create'}
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setShowCreateForm(false);
                setNewInstanceName('');
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {instances.length === 0 && !showCreateForm && (
        <div className="mt-4 pt-4 border-t text-center">
          <p className="text-sm text-muted-foreground mb-2">
            No instances found. Create your first instance to get started.
          </p>
          <Button size="sm" onClick={() => setShowCreateForm(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Create Instance
          </Button>
        </div>
      )}
    </Card>
  );
}
