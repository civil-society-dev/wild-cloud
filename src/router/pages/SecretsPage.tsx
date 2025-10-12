import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Label } from '../../components/ui/label';
import { Skeleton } from '../../components/ui/skeleton';
import { SecretInput } from '../../components/SecretInput';
import { Key, AlertTriangle, Save, X } from 'lucide-react';
import { useSecrets, useUpdateSecrets } from '../../hooks/useSecrets';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../components/ui/dialog';

export function SecretsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [isEditing, setIsEditing] = useState(false);
  const [editedSecrets, setEditedSecrets] = useState<Record<string, unknown>>({});
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);

  const { data: secrets, isLoading } = useSecrets(instanceId, true);
  const updateMutation = useUpdateSecrets(instanceId);

  const handleEdit = () => {
    setEditedSecrets(secrets || {});
    setIsEditing(true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setEditedSecrets({});
  };

  const handleSave = () => {
    setShowConfirmDialog(true);
  };

  const confirmSave = async () => {
    await updateMutation.mutateAsync(editedSecrets);
    setShowConfirmDialog(false);
    setIsEditing(false);
    setEditedSecrets({});
  };

  const handleSecretChange = (path: string, value: string) => {
    setEditedSecrets((prev) => {
      const updated = { ...prev };
      // Support nested paths using dot notation
      const keys = path.split('.');
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let current: any = updated;
      for (let i = 0; i < keys.length - 1; i++) {
        if (!current[keys[i]]) current[keys[i]] = {};
        current = current[keys[i]];
      }
      current[keys[keys.length - 1]] = value;
      return updated;
    });
  };

  // Flatten nested object into dot-notation paths
  const flattenSecrets = (obj: Record<string, unknown>, prefix = ''): Array<{ path: string; value: string }> => {
    const result: Array<{ path: string; value: string }> = [];
    for (const [key, value] of Object.entries(obj)) {
      const path = prefix ? `${prefix}.${key}` : key;
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        result.push(...flattenSecrets(value as Record<string, unknown>, path));
      } else {
        result.push({ path, value: String(value || '') });
      }
    }
    return result;
  };

  const getValue = (obj: Record<string, unknown>, path: string): string => {
    const keys = path.split('.');
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let current: any = obj;
    for (const key of keys) {
      if (!current || typeof current !== 'object') return '';
      current = current[key];
    }
    return String(current || '');
  };

  if (!instanceId) {
    return (
      <div className="flex items-center justify-center h-96">
        <Card className="p-6">
          <div className="flex items-center gap-3 text-muted-foreground">
            <AlertTriangle className="h-5 w-5" />
            <p>No instance selected</p>
          </div>
        </Card>
      </div>
    );
  }

  const secretsList = secrets ? flattenSecrets(secrets) : [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Secrets Management</h2>
          <p className="text-muted-foreground">
            Manage instance secrets securely
          </p>
        </div>
        {!isEditing ? (
          <Button onClick={handleEdit} disabled={isLoading}>
            Edit Secrets
          </Button>
        ) : (
          <div className="flex gap-2">
            <Button onClick={handleCancel} variant="outline">
              <X className="h-4 w-4" />
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateMutation.isPending}>
              <Save className="h-4 w-4" />
              Save Changes
            </Button>
          </div>
        )}
      </div>

      {isEditing && (
        <Card className="border-yellow-500 bg-yellow-50 dark:bg-yellow-950/20">
          <CardContent className="pt-6">
            <div className="flex gap-3">
              <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-500 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-medium text-yellow-900 dark:text-yellow-200">
                  Security Warning
                </p>
                <p className="text-sm text-yellow-800 dark:text-yellow-300 mt-1">
                  You are editing sensitive secrets. Make sure you are in a secure environment.
                  Changes will be saved immediately and cannot be undone.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Instance Secrets
          </CardTitle>
          <CardDescription>
            {isEditing ? 'Edit secret values below' : 'View encrypted secrets for this instance'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : secretsList.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Key className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">No secrets found</p>
              <p className="text-xs mt-1">Secrets will appear here once configured</p>
            </div>
          ) : (
            <div className="space-y-4">
              {secretsList.map(({ path, value }) => (
                <div key={path} className="space-y-2">
                  <Label htmlFor={path}>{path}</Label>
                  <SecretInput
                    value={isEditing ? getValue(editedSecrets, path) : value}
                    onChange={isEditing ? (newValue) => handleSecretChange(path, newValue) : undefined}
                    readOnly={!isEditing}
                  />
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Save</DialogTitle>
            <DialogDescription>
              Are you sure you want to save these secret changes? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowConfirmDialog(false)}>
              Cancel
            </Button>
            <Button onClick={confirmSave} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
