import Spinner from 'hew/Spinner';
import { ComponentPropsWithoutRef, FC, lazy, Suspense } from 'react';

// eslint-disable-next-line no-restricted-imports
const HewCodeEditor = lazy(() => import('hew/CodeEditor'));

export const CodeEditor: FC<ComponentPropsWithoutRef<typeof HewCodeEditor>> = (props) => (
  <Suspense fallback={<Spinner spinning tip="Loading code viewer..." />}>
    <HewCodeEditor {...props} />
  </Suspense>
);

export default CodeEditor;
