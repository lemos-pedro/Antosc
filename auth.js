function checkAuth() {
    const token = localStorage.getItem("access_token");

    if (!token) {
        // Sem token → volta ao login
        window.location.href = "login.html";
        return;
    }

    fetch("http://localhost:8000/api/auth/me", {
        method: "GET",
        headers: {
            "Authorization": "Bearer " + token
        }
    })
    .then(res => {
        if (!res.ok) {
            throw new Error("Token inválido");
        }
        return res.json();
    })
    .then(user => {
        console.log("Usuário autenticado:", user);
        // podes exibir user.nome no header do dashboard
        document.getElementById("userName").innerText = user.nome;
    })
    .catch(err => {
        console.error(err);
        // Token inválido → apaga e volta ao login
        localStorage.removeItem("access_token");
        window.location.href = "login.html";
    });
}

// Executar ao carregar a página
document.addEventListener("DOMContentLoaded", checkAuth);

