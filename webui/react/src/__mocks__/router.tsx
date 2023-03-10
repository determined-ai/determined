const router = (): { navigate: (path: string) => void } => ({
  navigate: (path: string): void => {
    global.window.history.pushState({}, '', path);
  },
});

export default router;
