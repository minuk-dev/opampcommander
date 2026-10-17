// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';
// Use React's timer fallback: the scaffold's Node MessageChannel keeps Jest alive.
delete global.MessageChannel;
