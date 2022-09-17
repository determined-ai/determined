import React from 'react';

import useWebSettings, { ProjectDetailType } from 'recoil/userSettings/useWebSettings';

const Test2: React.FC = () => {
  const [ archived, setArchived ] = useWebSettings(ProjectDetailType.Archived);
  const onClick = () => setArchived({ pd_archived: false });
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(archived.pd_archived)}</div>
    </>
  );
};

export default Test2;
