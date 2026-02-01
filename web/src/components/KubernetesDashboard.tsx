import { useState } from "react";
import { useParams } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { Button } from "./ui";
import { Check, ExternalLink, Copy } from "lucide-react";
import { useDashboardToken } from "../services/api/hooks/useUtilities";
import { useInstance } from "../services/api";

export function KubernetesDashboard() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [copied, setCopied] = useState(false);
  const { data: instance } = useInstance(instanceId || '');
  const { data: dashboardToken, isLoading: tokenLoading } = useDashboardToken(instanceId || '');

  const handleCopyToken = async () => {
    if (dashboardToken?.token) {
      try {
        // Use different methods based on browser compatibility
        if (navigator.clipboard && window.isSecureContext) {
          // Modern clipboard API (requires HTTPS or localhost)
          await navigator.clipboard.writeText(dashboardToken.token);
        } else {
          // Fallback method for older browsers or non-secure contexts
          const textArea = document.createElement("textarea");
          textArea.value = dashboardToken.token;
          textArea.style.position = "fixed";
          textArea.style.left = "-999999px";
          textArea.style.top = "-999999px";
          document.body.appendChild(textArea);
          textArea.focus();
          textArea.select();
          const successful = document.execCommand('copy');
          textArea.remove();
          if (!successful) {
            throw new Error('Fallback copy failed');
          }
        }
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      } catch (err) {
        console.error('Failed to copy token:', err);
        // Show user-friendly error message
        alert('Failed to copy token to clipboard. Please copy manually: ' + dashboardToken.token);
      }
    }
  };

  const handleOpenDashboard = () => {
    // Build dashboard URL from instance config
    // Dashboard is available at: https://dashboard.{cloud.internalDomain}
    const internalDomain = instance?.config?.cloud?.internalDomain;
    const dashboardUrl = internalDomain
      ? `https://dashboard.${internalDomain}`
      : 'https://dashboard.internal.wild.cloud';
    window.open(dashboardUrl, '_blank');
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Kubernetes Dashboard</h2>
          <p className="text-muted-foreground">
            Access the Kubernetes dashboard for advanced cluster management
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Access Dashboard</CardTitle>
          <CardDescription>
            Open the dashboard and copy your authentication token
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Button onClick={handleOpenDashboard} disabled={!instance}>
              <ExternalLink className="h-4 w-4 mr-2" />
              Open Dashboard
            </Button>
            <Button
              variant="outline"
              onClick={handleCopyToken}
              disabled={tokenLoading || !dashboardToken?.token}
            >
              {copied ? (
                <>
                  <Check className="h-4 w-4 mr-2" />
                  Copied!
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4 mr-2" />
                  Copy Token
                </>
              )}
            </Button>
          </div>
          {instance?.config?.cloud?.internalDomain && (
            <p className="text-xs text-muted-foreground mt-3">
              Dashboard URL: https://dashboard.{instance.config.cloud.internalDomain}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
