import React from 'react';

import useWebSettings, { ProjectDetailKey } from 'recoil/userSettings/useWebSettings';

const Test2: React.FC = () => {
  const [ archived, setArchived ] = useWebSettings(ProjectDetailKey.Archived);
  const onClick = () => setArchived({ archived: false });
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(archived.archived)}</div>
    </>
  );
};

export default Test2;
