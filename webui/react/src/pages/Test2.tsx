import React from 'react';

import useWebSettings, { ProjectDetailKey } from 'recoil/userSettings/useWebSettings';

const Test2: React.FC = () => {
  const [ pinned, setPinned ] = useWebSettings(ProjectDetailKey.Pinned);
  const onClick = () => setPinned({ pinned: [] });
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(pinned)}</div>
    </>
  );
};

export default Test2;
