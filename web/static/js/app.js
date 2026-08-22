// app.js — minimal client-side glue for the server-rendered pages. Each
// handler reads its data attributes off the DOM and calls the JSON API,
// keeping page logic in one small file.
//
// Auth: the JWT lives in the HttpOnly cas_token cookie set on login. It is
// never visible to JS (XSS can't exfiltrate it). Same-origin fetch attaches
// the cookie automatically, and the API Auth middleware accepts the cookie,
// so no Authorization header is needed here. The cookie is the auth.

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  return res.json().catch(() => ({}));
}

// Course form (course_form.html)
const form = document.getElementById("course-form");
if (form) {
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const msg = document.getElementById("form-msg");
    const data = {
      title: form.title.value,
      description: form.description.value,
      teacher_id: form.teacher_id.value,
      date: form.date.value,
      start_time: form.start_time.value,
      end_time: form.end_time.value,
      location: form.location.value,
      repeat_weekly: form.repeat_weekly.checked,
      count: parseInt(form.count.value || "1", 10),
      end_date: form.end_date.value,
    };
    const r = await api("POST", "/api/v1/courses/batch", data);
    if (r.success) {
      msg.textContent = "创建成功，正在跳转…";
      msg.className = "text-sm text-green-600";
      setTimeout(() => (location.href = "/"), 600);
    } else {
      msg.textContent = r.error || "创建失败";
      msg.className = "text-sm text-red-500";
    }
  });
}

// QR image (course_detail.html)
const qrImg = document.getElementById("qr-img");
if (qrImg) {
  const id = document.getElementById("refresh-qr").dataset.course;
  const load = () => (qrImg.src = `/api/v1/courses/${id}/qr/image?t=${Date.now()}`);
  load();
  document.getElementById("refresh-qr").addEventListener("click", load);
}

// Check-in (checkin.html)
const checkinBtn = document.getElementById("checkin-btn");
if (checkinBtn) {
  checkinBtn.addEventListener("click", async () => {
    const status = document.getElementById("checkin-status");
    const r = await api("POST", `/api/v1/courses/${checkinBtn.dataset.course}/checkin`, {
      token: checkinBtn.dataset.token,
    });
    if (r.success) {
      status.textContent = "签到成功 ✓";
      status.className = "text-green-600";
    } else {
      status.textContent = r.error || "签到失败";
      status.className = "text-red-500";
    }
  });
}

// Student import (students.html)
const importBtn = document.getElementById("import-btn");
if (importBtn) {
  const modal = document.getElementById("import-modal");
  importBtn.addEventListener("click", () => modal.classList.remove("hidden"));
  document.getElementById("cancel-import").addEventListener("click", () => modal.classList.add("hidden"));
  document.getElementById("submit-import").addEventListener("click", async () => {
    const csv = document.getElementById("csv-input").value;
    const msg = document.getElementById("import-msg");
    const rows = csv.split(/\r?\n/).filter(Boolean).map((line) => {
      const [name, email, phone] = line.split(",").map((s) => s.trim());
      return { name, email, phone };
    });
    const r = await api("POST", "/api/v1/students/import", { students: rows });
    if (r.success) {
      msg.textContent = `导入完成：${r.data.imported} 名学员`;
      msg.className = "text-sm text-green-600";
      setTimeout(() => location.reload(), 1000);
    } else {
      msg.textContent = r.error || "导入失败";
      msg.className = "text-sm text-red-500";
    }
  });
}

// Teacher management (teachers.html)
const newBtn = document.getElementById("new-btn");
if (newBtn) {
  const editModal = document.getElementById("edit-modal");
  const pwModal = document.getElementById("pw-modal");
  let editingId = "";

  const openNew = () => {
    editingId = "";
    document.getElementById("edit-title").textContent = "新增讲师";
    document.getElementById("f-name").value = "";
    document.getElementById("f-email").value = "";
    document.getElementById("f-phone").value = "";
    document.getElementById("f-password").value = "";
    document.getElementById("f-password").parentElement.classList.remove("hidden");
    document.getElementById("edit-msg").textContent = "";
    editModal.classList.remove("hidden");
  };
  const openEdit = (row) => {
    editingId = row.dataset.id;
    document.getElementById("edit-title").textContent = "编辑讲师";
    document.getElementById("f-name").value = row.querySelector(".t-name").textContent;
    document.getElementById("f-email").value = row.querySelector(".t-email").textContent;
    document.getElementById("f-phone").value = row.querySelector(".t-phone").textContent;
    document.getElementById("f-password").value = "";
    document.getElementById("f-password").parentElement.classList.add("hidden");
    document.getElementById("edit-msg").textContent = "";
    editModal.classList.remove("hidden");
  };

  newBtn.addEventListener("click", openNew);
  document.getElementById("cancel-edit").addEventListener("click", () => editModal.classList.add("hidden"));
  document.getElementById("submit-edit").addEventListener("click", async () => {
    const msg = document.getElementById("edit-msg");
    const body = {
      name: document.getElementById("f-name").value.trim(),
      email: document.getElementById("f-email").value.trim(),
      phone: document.getElementById("f-phone").value.trim(),
    };
    const pw = document.getElementById("f-password").value;
    let r;
    if (editingId) {
      r = await api("PUT", `/api/v1/teachers/${editingId}`, body);
    } else {
      body.password = pw;
      r = await api("POST", "/api/v1/teachers", body);
    }
    if (r.success) {
      msg.textContent = "已保存，正在刷新…";
      msg.className = "text-sm text-green-600";
      setTimeout(() => location.reload(), 600);
    } else {
      msg.textContent = r.error || "保存失败";
      msg.className = "text-sm text-red-500";
    }
  });

  document.querySelectorAll(".edit-btn").forEach((b) =>
    b.addEventListener("click", (e) => openEdit(e.target.closest("tr"))));

  document.querySelectorAll(".pw-btn").forEach((b) =>
    b.addEventListener("click", (e) => {
      editingId = e.target.closest("tr").dataset.id;
      document.getElementById("f-newpw").value = "";
      document.getElementById("pw-msg").textContent = "";
      pwModal.classList.remove("hidden");
    }));
  document.getElementById("cancel-pw").addEventListener("click", () => pwModal.classList.add("hidden"));
  document.getElementById("submit-pw").addEventListener("click", async () => {
    const msg = document.getElementById("pw-msg");
    const r = await api("PATCH", `/api/v1/teachers/${editingId}/password`, {
      new_password: document.getElementById("f-newpw").value,
    });
    if (r.success) {
      msg.textContent = "密码已重置";
      msg.className = "text-sm text-green-600";
      setTimeout(() => pwModal.classList.add("hidden"), 800);
    } else {
      msg.textContent = r.error || "重置失败";
      msg.className = "text-sm text-red-500";
    }
  });

  document.querySelectorAll(".del-btn").forEach((b) =>
    b.addEventListener("click", async (e) => {
      const id = e.target.closest("tr").dataset.id;
      if (!confirm("确认删除该讲师？")) return;
      const r = await api("DELETE", `/api/v1/teachers/${id}`);
      if (r.success) {
        location.reload();
      } else {
        alert(r.error || "删除失败");
      }
    }));
}
