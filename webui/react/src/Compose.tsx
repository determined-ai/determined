import React, { JSXElementConstructor, PropsWithChildren, ReactNode } from 'react';

interface Props {
  children: React.ReactNode;
  components: JSXElementConstructor<PropsWithChildren<{children: ReactNode}>>[];
}

const Compose = (props: Props): JSX.Element => {
  const { components = [], children } = props;

  return (
    <>
      {components.reduceRight((acc, Comp) => {
        return <Comp>{acc}</Comp>;
      }, children)}
    </>
  );
};

export default Compose;
