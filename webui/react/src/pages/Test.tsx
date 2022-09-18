import React, { useState } from 'react';

import useWebSettings, { UserWebSettingsKeys } from 'recoil/userSettings/useWebSettings';

const Test: React.FC = () => {
  const [ count, setCount ] = useState<number>(0);
  const [ pinned, setPinned ] = useWebSettings(UserWebSettingsKeys.PG_Pinned);

  const onClick = () => {
    setPinned({ pd_pinned: { 1: [ 1, 2, count + 1 ] } });
    setCount((prev) => prev + 1);
  };
  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>{JSON.stringify(pinned.pd_pinned)}</div>
    </>
  );
};

export default Test;
