import { RouterProvider } from 'react-router';
import { router } from './router';
import { Toaster } from 'sonner';

function App() {
  return (
    <>
      <RouterProvider router={router} />
      <Toaster />
    </>
  );
}

export default App;
