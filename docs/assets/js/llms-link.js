// Open llms.txt link in a new tab
document.addEventListener("DOMContentLoaded", function () {
  const links = document.querySelectorAll('a[href$="llms.txt"]');
  links.forEach(function (link) {
    link.setAttribute("target", "_blank");
    link.setAttribute("rel", "noopener");
  });
});
