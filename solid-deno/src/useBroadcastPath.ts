import type { BroadcastPath } from "@okdaichi/moq";
import { useUser } from "./user/context.ts";

export function useBroadcastPath() {
	const user = useUser();
	const broadcastPath: BroadcastPath = `/${user.name()}`;
	return broadcastPath;
}
