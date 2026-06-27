export interface Connection {
  id: string;
  instanceUid: string;
  namespace: string;
  type: string;
  /**
   * The server instance holding the connection. Populated only for cluster-scoped
   * listings (scope=cluster); empty for the node-local listing.
   */
  serverId?: string;
  lastCommunicatedAt: string;
  alive: boolean;
}
