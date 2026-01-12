import { useParams } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { TerminalSquare } from "lucide-react";
import { XTerminal } from "./XTerminal";

export function Terminal() {
  const { instanceId } = useParams<{ instanceId: string }>();

  if (!instanceId) {
    return (
      <Card className="flex flex-col h-full">
        <CardHeader className="shrink-0">
          <CardTitle className="flex items-center gap-2">
            <TerminalSquare className="h-5 w-5" />
            Terminal
          </CardTitle>
          <CardDescription>
            No instance selected
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="shrink-0 pb-2">
        <CardTitle className="flex items-center gap-2">
          <TerminalSquare className="h-5 w-5" />
          Terminal
        </CardTitle>
        <CardDescription>
          Interactive shell on Wild Central - {instanceId}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-1 min-h-0 pb-4">
        <XTerminal key={instanceId} instanceId={instanceId} />
      </CardContent>
    </Card>
  );
}
