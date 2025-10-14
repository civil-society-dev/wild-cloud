import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Loader2, Save } from 'lucide-react';
import { useServiceConfig, useServiceStatus } from '@/hooks/useServices';
import type { ServiceManifest } from '@/services/api/types';

interface ServiceConfigEditorProps {
  instanceName: string;
  serviceName: string;
  manifest?: ServiceManifest;
  onClose: () => void;
  onSuccess?: () => void;
}

export function ServiceConfigEditor({
  instanceName,
  serviceName,
  manifest: _manifestProp, // Ignore the prop, fetch from status instead
  onClose,
  onSuccess,
}: ServiceConfigEditorProps) {
  const { config, isLoading: configLoading, updateConfig, isUpdating } = useServiceConfig(instanceName, serviceName);
  const { data: statusData, isLoading: statusLoading } = useServiceStatus(instanceName, serviceName);

  // Use manifest from status endpoint which includes full serviceConfig
  const manifest = statusData?.manifest;
  const [formData, setFormData] = useState<Record<string, unknown>>({});
  const [redeploy, setRedeploy] = useState(true);
  const [fetch, setFetch] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Initialize form data when config loads
  useEffect(() => {
    if (config) {
      setFormData(config);
    }
  }, [config]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    try {
      await updateConfig({ config: formData, redeploy, fetch });
      setSuccess(true);
      if (onSuccess) {
        setTimeout(() => {
          onSuccess();
        }, 1500);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update configuration');
    }
  };

  const handleInputChange = (key: string, value: string) => {
    setFormData((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const getDisplayValue = (value: unknown): string => {
    if (value === null || value === undefined) return '';
    if (typeof value === 'object') {
      return JSON.stringify(value, null, 4);
    }
    return String(value);
  };

  const isObjectValue = (value: unknown): boolean => {
    return value !== null && value !== undefined && typeof value === 'object';
  };

  const isLoading = configLoading || statusLoading;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Get configurable keys from serviceConfig definitions
  const configKeys = manifest?.serviceConfig
    ? Object.keys(manifest.serviceConfig).map(key => manifest.serviceConfig![key].path)
    : [];

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Edit Service Configuration</h2>
        <p className="text-sm text-muted-foreground">{serviceName}</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
          {configKeys.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">
              No configuration options available for this service.
            </p>
          ) : (
            configKeys.map((key) => {
              const value = formData[key];
              const isObject = isObjectValue(value);

              // Find the config definition for this path
              const configDef = manifest?.serviceConfig
                ? Object.values(manifest.serviceConfig).find(def => def.path === key)
                : undefined;

              return (
                <div key={key} className="space-y-2">
                  <Label htmlFor={key}>
                    {configDef?.prompt || key}
                    {configDef?.default && (
                      <span className="text-xs text-muted-foreground ml-2">
                        (default: {configDef.default})
                      </span>
                    )}
                  </Label>
                  {isObject ? (
                    <Textarea
                      id={key}
                      value={getDisplayValue(value)}
                      onChange={(e) => handleInputChange(key, e.target.value)}
                      placeholder={configDef?.default || ''}
                      rows={5}
                      className="font-mono text-sm"
                    />
                  ) : (
                    <Input
                      id={key}
                      value={getDisplayValue(value)}
                      onChange={(e) => handleInputChange(key, e.target.value)}
                      placeholder={configDef?.default || ''}
                    />
                  )}
                </div>
              );
            })
          )}

          <div className="space-y-2 pt-4 border-t">
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="redeploy-checkbox"
                checked={redeploy}
                onChange={(e) => setRedeploy(e.target.checked)}
                className="rounded"
              />
              <Label htmlFor="redeploy-checkbox" className="cursor-pointer">
                Redeploy service after updating configuration
              </Label>
            </div>
            {redeploy && (
              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="fetch-checkbox"
                  checked={fetch}
                  onChange={(e) => setFetch(e.target.checked)}
                  className="rounded"
                />
                <Label htmlFor="fetch-checkbox" className="cursor-pointer">
                  Fetch fresh templates from directory before redeploying
                </Label>
              </div>
            )}
          </div>

          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          {success && (
            <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-600 dark:text-green-400">
              Configuration updated successfully!
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 pt-4 border-t">
          <Button type="button" variant="outline" onClick={onClose} disabled={isUpdating}>
            Cancel
          </Button>
          <Button type="submit" disabled={isUpdating}>
            {isUpdating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                Saving...
              </>
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </>
            )}
          </Button>
        </div>
      </form>
    </div>
  );
}
