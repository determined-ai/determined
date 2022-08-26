import React from 'react';

import ActionDropdown from './ActionDropdown';

export default {
  component: ActionDropdown,
  title: 'ActionDropdown',
};

const FIRST_ACTION = 'First Action';
const SECOND_ACTION = 'Second Action';
const THIRD_ACTION = 'Third Action';

const actions = [
  FIRST_ACTION,
  SECOND_ACTION,
  THIRD_ACTION,
];

const disabled = {
  [FIRST_ACTION]: false,
  [SECOND_ACTION]: true,
  [THIRD_ACTION]: false,
};

const triggers = {
  [FIRST_ACTION]: () => { return; },
  [SECOND_ACTION]: () => { return; },
  [THIRD_ACTION]: () => { return; },
};

export const Default = (): React.ReactNode => (
  <ActionDropdown
    actionOrder={actions}
    id="id"
    kind="kind"
    onError={() => { return; }}
    onTrigger={triggers}
  />
);

export const DisabledAction = (): React.ReactNode => (
  <ActionDropdown
    actionOrder={actions}
    disabled={disabled}
    id="id"
    kind="kind"
    onError={() => { return; }}
    onTrigger={triggers}
  />
);
