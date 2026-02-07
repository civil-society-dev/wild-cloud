import { useParams } from "react-router-dom";
import { XTerminal } from "./XTerminal";

export function Terminal() {
  const { instanceId } = useParams<{ instanceId: string }>();

  if (!instanceId) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-3xl font-bold tracking-tight">Terminal</h2>
            <p className="text-muted-foreground">No instance selected</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Terminal</h2>
          <p className="text-muted-foreground">
            Interactive shell on Wild Central
          </p>
        </div>
      </div>

      <XTerminal key={instanceId} instanceId={instanceId} />
    </div>
  );
}
