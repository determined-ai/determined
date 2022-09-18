import React from 'react';

import useWebSettings, { UserWebSettingsKeys } from 'recoil/userSettings/useWebSettings';

const Test2: React.FC = () => {
  const [ archived, setArchived ] = useWebSettings(UserWebSettingsKeys.PG_Archived);
  const [ tableLimit, setTabeLimit ] = useWebSettings(UserWebSettingsKeys.PG_TableLimit);
  const onClick = () => {
    setTabeLimit({ pd_tableLimit: 20 });
    setArchived({ pd_archived: !archived.pd_archived });
  };
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{archived.pd_archived ? 'true!!' : 'false!!'}</div>
      <div>{tableLimit.pd_tableLimit}</div>
    </>
  );
};

export default Test2;
