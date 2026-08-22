// Package templates loads the HTML templates from disk into the gin
// template engine. It loads the shared partials (head/nav/foot) together
// with every page into one template tree; each page is fully self-contained
// (it invokes the partials) so any page can be rendered by filename
// without per-request cloning or template-name collisions.
package templates

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Load parses the partials and every page template from dir and registers
// the combined tree on the engine.
func Load(r *gin.Engine, dir string) error {
	files := []string{
		filepath.Join(dir, "partials", "head.html"),
		filepath.Join(dir, "partials", "nav.html"),
		filepath.Join(dir, "partials", "foot.html"),
	}
	pages := []string{
		"login.html", "register.html", "index.html", "course_detail.html",
		"course_form.html", "my_schedule.html", "checkin.html",
		"attendance.html", "students.html", "teachers.html",
	}
	for _, p := range pages {
		files = append(files, filepath.Join(dir, p))
	}
	r.LoadHTMLFiles(files...)
	return nil
}
