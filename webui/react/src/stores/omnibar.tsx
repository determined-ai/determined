import React, {
  createContext,
  Dispatch,
  PropsWithChildren,
  SetStateAction,
  useContext,
  useState,
} from 'react';

import { noOp } from 'shared/utils/service';

type OmnibarContext = {
  isShowing: boolean;
  updateShowing: Dispatch<SetStateAction<boolean>>;
};

const OmnibarContext = createContext<OmnibarContext>({
  isShowing: false,
  updateShowing: noOp,
});

export const OmnibarProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<boolean>(false);

  return (
    <OmnibarContext.Provider value={{ isShowing: state, updateShowing: setState }}>
      {children}
    </OmnibarContext.Provider>
  );
};

export const useOmnibarContext = (): OmnibarContext => {
  return useContext(OmnibarContext);
};
