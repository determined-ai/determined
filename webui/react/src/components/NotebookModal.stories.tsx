import React from 'react';

import NotebookModal from './NotebookModal';

export default {
  component: NotebookModal,
  title: 'NotebookModal',
};

export const Default = (): React.ReactNode => {
  return <NotebookModal forceVisible={true} />;
};
