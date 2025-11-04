import { Alert } from '../ui/alert';
import { AlertCircle } from 'lucide-react';

interface TroubleshootingPanelProps {
  step: number;
}

const TROUBLESHOOTING_STEPS: Record<number, string[]> = {
  1: [
    'Check etcd service status with: talosctl -n <node-ip> service etcd',
    'View etcd logs: talosctl -n <node-ip> logs etcd',
    'Verify bootstrap completed successfully',
  ],
  2: [
    'Check VIP controller logs: kubectl logs -n kube-system -l k8s-app=kube-vip',
    'Verify network configuration allows VIP assignment',
    'Check that VIP range is configured correctly in cluster config',
  ],
  3: [
    'Check kubelet logs: talosctl -n <node-ip> logs kubelet',
    'Verify static pod manifests: talosctl -n <node-ip> list /etc/kubernetes/manifests',
    'Try restarting kubelet: talosctl -n <node-ip> service kubelet restart',
  ],
  4: [
    'Check API server logs: kubectl logs -n kube-system kube-apiserver-<node>',
    'Verify API server is running: talosctl -n <node-ip> service kubelet',
    'Test API server on node IP: curl -k https://<node-ip>:6443/healthz',
  ],
  5: [
    'Check API server logs for connection errors',
    'Test API server on node IP first: curl -k https://<node-ip>:6443/healthz',
    'Verify network connectivity to VIP address',
  ],
  6: [
    'Check kubelet logs: talosctl -n <node-ip> logs kubelet',
    'Verify API server is accessible: kubectl get nodes',
    'Check network connectivity between node and API server',
  ],
};

export function TroubleshootingPanel({ step }: TroubleshootingPanelProps) {
  const steps = TROUBLESHOOTING_STEPS[step] || [
    'Check logs for detailed error information',
    'Verify network connectivity',
    'Ensure all prerequisites are met',
  ];

  return (
    <Alert variant="error" className="mt-4">
      <AlertCircle className="h-4 w-4" />
      <div>
        <strong className="font-semibold">Troubleshooting Steps</strong>
        <ul className="mt-2 ml-4 list-disc space-y-1 text-sm">
          {steps.map((troubleshootingStep, index) => (
            <li key={index}>{troubleshootingStep}</li>
          ))}
        </ul>
      </div>
    </Alert>
  );
}
