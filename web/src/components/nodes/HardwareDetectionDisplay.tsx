import type { HardwareInfo } from '../../services/api/types';

interface HardwareDetectionDisplayProps {
  detection: HardwareInfo;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function HardwareDetectionDisplay({ detection }: HardwareDetectionDisplayProps) {
  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-4 space-y-3">
      <div className="flex items-start">
        <svg
          className="h-5 w-5 text-green-500 mt-0.5 mr-3 flex-shrink-0"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-700 dark:text-gray-300">IP Address</p>
          <p className="text-sm text-gray-900 dark:text-gray-100 font-mono">{detection.ip}</p>
        </div>
      </div>

      {detection.interface && (
        <div className="flex items-start">
          <svg
            className="h-5 w-5 text-green-500 mt-0.5 mr-3 flex-shrink-0"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
              clipRule="evenodd"
            />
          </svg>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Network Interface</p>
            <p className="text-sm text-gray-900 dark:text-gray-100 font-mono">{detection.interface}</p>
          </div>
        </div>
      )}

      {detection.disks && detection.disks.length > 0 && (
        <div className="flex items-start">
          <svg
            className="h-5 w-5 text-green-500 mt-0.5 mr-3 flex-shrink-0"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
              clipRule="evenodd"
            />
          </svg>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Available Disks</p>
            <ul className="mt-1 space-y-1">
              {detection.disks.map((disk) => (
                <li key={disk.path} className="text-sm text-gray-900 dark:text-gray-100">
                  <span className="font-mono">{disk.path}</span>
                  {disk.size > 0 && (
                    <span className="text-gray-500 dark:text-gray-400 ml-2">
                      ({formatBytes(disk.size)})
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  );
}
