'use client';

import type { Condition } from '@shared/api';
import Badge from './Badge';
import Tooltip from './Tooltip';

interface Props {
  conditions: Condition[] | undefined;
  emptyText?: string;
}

// Conditions render the same way everywhere: True is good, False is a warning,
// Unknown is neutral, and the reason/message hides in the tooltip.
export default function ConditionBadges({ conditions, emptyText = '-' }: Props) {
  if (!conditions || conditions.length === 0) {
    return <span className="text-muted-foreground">{emptyText}</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {conditions.map((c, i) => (
        <Tooltip key={`${c.type}-${i}`} content={[c.reason, c.message].filter(Boolean).join(' — ')}>
          <Badge
            variant={c.status === 'True' ? 'success' : c.status === 'False' ? 'warning' : 'outline'}
          >
            {c.type}: {c.status}
          </Badge>
        </Tooltip>
      ))}
    </div>
  );
}
