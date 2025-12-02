import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { FunctionList } from './pages/FunctionList';
import { FunctionDetail } from './pages/FunctionDetail';
import { FunctionCreate } from './pages/FunctionCreate';
import './App.css';

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <header className="header">
          <a href="/functions" className="logo">SoftGate</a>
        </header>

        <main className="main-content">
          <Routes>
            <Route path="/" element={<Navigate to="/functions" replace />} />
            <Route path="/functions" element={<FunctionList />} />
            <Route path="/functions/new" element={<FunctionCreate />} />
            <Route path="/functions/:id" element={<FunctionDetail />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
